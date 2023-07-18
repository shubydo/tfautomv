package planner

import (
	"fmt"
	"os"
	"os/exec"
)

// An Option configures a Planner's behavior.
type Option func(*settings)

// WithWorkdir sets the directory where the Planner will run Terraform commands.
// If this option is not provided, the Planner runs in the current working
// directory.
func WithWorkdir(workdir string) Option {
	return func(s *settings) {
		s.workdir = workdir
	}
}

// WithTerraformBin overrides the `terraform` binary that the Planner uses.
// If this option is not provided, the Planner uses the `terraform` binary it
// finds in the PATH.
func WithTerraformBin(path string) Option {
	return func(s *settings) {
		s.terraformBin = path
	}
}

// WithSkipInit configures whether the Planner skips initializing the module
// before obtaining a plan from Terraform. By default, the Planner does not skip
// this step.
//
// Skipping the init step can save time, but subsequent steps may fail if the
// module was not initialized before using the Planner.
func WithSkipInit(skipInit bool) Option {
	return func(s *settings) {
		s.skipInit = skipInit
	}
}

// WithSkipRefresh configures whether the Planner skips refreshing the module's
// state before obtaining a plan from Terraform. By default, the Planner does
// not skip this step.
//
// Skipping the refresh step can save time, but can result in Terraform basing
// its plan on stale data.
func WithSkipRefresh(skipRefresh bool) Option {
	return func(s *settings) {
		s.skipRefresh = skipRefresh
	}
}

// settings hold the values of all planner options
type settings struct {
	workdir      string
	terraformBin string
	skipInit     bool
	skipRefresh  bool
}

func newSettings() *settings {
	return &settings{}
}

func (s *settings) apply(opts []Option) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *settings) validate() error {
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
