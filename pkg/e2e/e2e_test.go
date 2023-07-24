//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"

	"github.com/busser/tfautomv/pkg/slices"
)

// ANSI escape sequence used for color output
const colorEscapeSequence = "\x1b"

func TestE2E(t *testing.T) {
	tests := []struct {
		name    string
		workdir string
		args    []string

		wantChanges int

		skip       bool
		skipReason string
	}{
		{
			name:        "same attributes",
			workdir:     filepath.Join("testdata", "same-attributes"),
			wantChanges: 0,
		},
		{
			name:        "requires dependency analysis",
			workdir:     filepath.Join("testdata", "requires-dependency-analysis"),
			wantChanges: 0,
			skip:        true,
			skipReason:  "tfautomv cannot yet solve this case",
		},
		{
			name:        "same type",
			workdir:     filepath.Join("testdata", "same-type"),
			wantChanges: 0,
		},
		{
			name:        "different attributes",
			workdir:     filepath.Join("testdata", "different-attributes"),
			wantChanges: 2,
		},
		{
			name:    "ignore different attributes",
			workdir: filepath.Join("testdata", "different-attributes"),
			args: []string{
				"-ignore=everything:random_pet:length",
			},
			wantChanges: 1,
		},
		{
			name:        "terraform cloud",
			workdir:     filepath.Join("testdata", "terraform-cloud"),
			wantChanges: 0,
			skip:        true,
			skipReason:  "tfautomv is currently incompatible with Terraform Cloud workspaces with the \"Remote\" execution mode.\nFor more details, see https://github.com/busser/tfautomv/issues/17",
		},
		{
			name:    "terragrunt",
			workdir: filepath.Join("testdata", "terragrunt"),
			args: []string{
				"-terraform-bin=terragrunt",
			},
			wantChanges: 0,
		},
	}

	binPath := buildBinary(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, outputFormat := range []string{"blocks", "commands"} {
				t.Run(outputFormat, func(t *testing.T) {

					originalWorkdir := filepath.Join(tt.workdir, "original-code")
					refactoredWorkdir := filepath.Join(tt.workdir, "refactored-code")

					terraformBin := "terraform"
					for _, a := range tt.args {
						if strings.HasPrefix(a, "-terraform-bin=") {
							terraformBin = strings.TrimPrefix(a, "-terraform-bin=")
						}
					}

					/*
						Skip tests that serve as documentation of known limitations or
						use features incompatible with the Terraform CLI's version.
					*/

					if tt.skip {
						t.Skip(tt.skipReason)
					}

					if outputFormat == "blocks" {
						tf, err := tfexec.NewTerraform(originalWorkdir, terraformBin)
						if err != nil {
							t.Fatal(err)
						}
						tfVer, _, err := tf.Version(context.TODO(), false)
						if err != nil {
							t.Fatalf("failed to get terraform version: %v", err)
						}

						if tfVer.LessThan(version.Must(version.NewVersion("1.1"))) {
							t.Skip("terraform moves output format is only supported in terraform 1.1 and above")
						}
					}

					/*
						Create a fresh environment for each test.
					*/

					setupWorkdir(t, originalWorkdir, refactoredWorkdir, terraformBin)

					args := append(tt.args, fmt.Sprintf("-output=%s", outputFormat))

					/*
						Run tfautomv to generate `moved` blocks or `terraform state mv` commands.
					*/

					tfautomvCmd := exec.Command(binPath, args...)
					tfautomvCmd.Dir = refactoredWorkdir

					var tfautomvStdout bytes.Buffer
					var tfautomvCompleteOutput bytes.Buffer
					tfautomvCmd.Stdout = io.MultiWriter(&tfautomvStdout, &tfautomvCompleteOutput, os.Stderr)
					tfautomvCmd.Stderr = io.MultiWriter(&tfautomvCompleteOutput, os.Stderr)

					if err := tfautomvCmd.Run(); err != nil {
						t.Fatalf("running tfautomv: %v", err)
					}

					/*
						If using `terraform state mv` commands, run them.
					*/

					if outputFormat == "commands" {
						cmd := exec.Command("/bin/sh")
						cmd.Dir = refactoredWorkdir

						cmd.Stdin = &tfautomvStdout
						cmd.Stdout = os.Stderr
						cmd.Stderr = os.Stderr

						if err := cmd.Run(); err != nil {
							t.Fatalf("running terraform state mv commands: %v", err)
						}
					}

					/*
						Count how many changes remain in Terraform's plan.
					*/

					tf, err := tfexec.NewTerraform(refactoredWorkdir, terraformBin)
					if err != nil {
						t.Fatal(err)
					}

					planFile, err := os.CreateTemp("", "tfautomv.*.plan")
					if err != nil {
						t.Fatal(err)
					}
					defer os.Remove(planFile.Name())
					if _, err := tf.Plan(context.TODO(), tfexec.Out(planFile.Name())); err != nil {
						t.Fatalf("terraform plan (after addings moves): %v", err)
					}
					plan, err := tf.ShowPlanFile(context.TODO(), planFile.Name())
					if err != nil {
						t.Fatalf("terraform show (after addings moves): %v", err)
					}

					changes := numChanges(plan)
					if changes != tt.wantChanges {
						t.Errorf("%d changes remaining, want %d", changes, tt.wantChanges)
					}
				})
			}
		})
	}
}

func numChanges(p *tfjson.Plan) int {
	count := 0

	for _, rc := range p.ResourceChanges {
		if slices.Contains(rc.Change.Actions, tfjson.ActionCreate) || slices.Contains(rc.Change.Actions, tfjson.ActionDelete) {
			count++
		}
	}

	return count
}

func buildBinary(t *testing.T) string {
	t.Helper()

	rootDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("could not get root directory: %v", err)
	}

	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = rootDir
	buildCmd.Stdout = os.Stderr
	buildCmd.Stderr = os.Stderr

	t.Log("Building tfautomv binary...")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("make build: %v", err)
	}

	binPath := filepath.Join(rootDir, "bin", "tfautomv")
	return binPath
}

func setupWorkdir(t *testing.T, originalWorkdir, refactoredWorkdir, terraformBin string) {
	t.Helper()

	filesToRemove := []string{
		filepath.Join(originalWorkdir, "terraform.tfstate"),
		filepath.Join(originalWorkdir, ".terraform.lock.hcl"),
		filepath.Join(refactoredWorkdir, "terraform.tfstate"),
		filepath.Join(refactoredWorkdir, ".terraform.lock.hcl"),
		filepath.Join(refactoredWorkdir, "moves.tf"),
	}
	for _, f := range filesToRemove {
		ensureFileRemoved(t, f)
	}

	directoriesToRemove := []string{
		filepath.Join(originalWorkdir, ".terraform"),
		filepath.Join(refactoredWorkdir, ".terraform"),
	}
	for _, d := range directoriesToRemove {
		ensureDirectoryRemoved(t, d)
	}

	original, err := tfexec.NewTerraform(originalWorkdir, terraformBin)
	if err != nil {
		t.Fatal(err)
	}

	if err := original.Init(context.TODO()); err != nil {
		t.Fatal(err)
	}
	if err := original.Apply(context.TODO()); err != nil {
		t.Fatal(err)
	}

	os.Rename(
		filepath.Join(originalWorkdir, "terraform.tfstate"),
		filepath.Join(refactoredWorkdir, "terraform.tfstate"),
	)
}

func ensureFileRemoved(t *testing.T, path string) {
	t.Helper()

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("could not remove file %q: %v", path, err)
	}
}

func ensureDirectoryRemoved(t *testing.T, path string) {
	t.Helper()

	err := os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("could not remove directory %q: %v", path, err)
	}
}
