package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/busser/tfautomv/pkg/pretty"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// GetPlan obtains a Terraform plan from the module in the given working
// directory. It does so by running a series of Terraform commands.
func GetPlan(ctx context.Context, workdir string, opts ...PlanOption) (*tfjson.Plan, error) {
	var settings planSettings

	settings.apply(append(defaultOptions(), opts...))

	err := settings.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid plan options: %w", err)
	}

	tf, err := tfexec.NewTerraform(workdir, settings.terraformBin)
	if err != nil {
		return nil, fmt.Errorf("failed to create Terraform executor: %w", err)
	}

	if !settings.skipInit {
		logCommand(workdir, "terraform init")

		err := tf.Init(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Terraform: %w", err)
		}
	}

	if !settings.skipRefresh {
		logCommand(workdir, "terraform refresh")

		err := tf.Refresh(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh Terraform state: %w", err)
		}
	}

	logCommand(workdir, "terraform plan")

	planFile, err := os.CreateTemp("", "tfautomv.*.plan")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file to store raw plan: %w", err)
	}
	defer os.Remove(planFile.Name())

	_, err = tf.Plan(ctx, tfexec.Out(planFile.Name()))
	if err != nil {
		return nil, fmt.Errorf("failed to compute Terraform plan: %w", err)
	}

	plan, err := tf.ShowPlanFile(ctx, planFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read raw Terraform plan: %w", err)
	}

	return plan, nil
}

type planSettings struct {
	workdir      string
	terraformBin string
	skipInit     bool
	skipRefresh  bool
}

// An PlanOption configures how Terraform's plan is fetched.
type PlanOption func(*planSettings)

func defaultOptions() []PlanOption {
	return []PlanOption{
		WithWorkdir("."),
		WithTerraformBin("terraform"),
	}
}

// WithWorkdir sets the directory where the Planner will run Terraform commands.
// If this option is not provided, the Planner runs in the current working
// directory.
func WithWorkdir(workdir string) PlanOption {
	return func(s *planSettings) {
		s.workdir = workdir
	}
}

// WithTerraformBin overrides the `terraform` binary that the Planner uses.
// If this option is not provided, the Planner uses the `terraform` binary it
// finds in the PATH.
func WithTerraformBin(path string) PlanOption {
	return func(s *planSettings) {
		s.terraformBin = path
	}
}

// WithSkipInit configures whether the Planner skips initializing the module
// before obtaining a plan from Terraform. By default, the Planner does not skip
// this step.
//
// Skipping the init step can save time, but subsequent steps may fail if the
// module was not initialized before using the Planner.
func WithSkipInit(skipInit bool) PlanOption {
	return func(s *planSettings) {
		s.skipInit = true
	}
}

// WithSkipRefresh configures whether the Planner skips refreshing the module's
// state before obtaining a plan from Terraform. By default, the Planner does
// not skip this step.
//
// Skipping the refresh step can save time, but can result in Terraform basing
// its plan on stale data.
func WithSkipRefresh(skipRefresh bool) PlanOption {
	return func(s *planSettings) {
		s.skipRefresh = true
	}
}

func (s *planSettings) apply(opts []PlanOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *planSettings) validate() error {
	if !isDirectory(s.workdir) {
		return fmt.Errorf("target directory %q not found", s.workdir)
	}

	if !isInPath(s.terraformBin) {
		return fmt.Errorf("executable %q not found in PATH", s.terraformBin)
	}

	return nil
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func isInPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func logCommand(workdir, command string) {
	if workdir == "." {
		workdir = "current directory"
	}
	os.Stderr.WriteString(pretty.Colorf("running [bold]%s[reset] in [bold]%s[reset]...", command, workdir) + "\n")
}
