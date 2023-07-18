package planner

import (
	"context"
	"fmt"
	"os"

	"github.com/busser/tfautomv/pkg/logger"
	"github.com/busser/tfautomv/pkg/planner/flatmap"
	"github.com/busser/tfautomv/pkg/slices"
	"github.com/busser/tfautomv/pkg/terraform"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// Planner obtains a Terraform plan from a directory.
type Planner struct {
	tf          *tfexec.Terraform
	skipInit    bool
	skipRefresh bool
}

func defaultOptions() []Option {
	return []Option{
		WithWorkdir("."),
		WithTerraformBin("terraform"),
		WithSkipInit(false),
		WithSkipRefresh(false),
	}
}

// New builds a new Planner configured with the provided options.
func New(opts ...Option) (*Planner, error) {
	s := newSettings()

	s.apply(append(defaultOptions(), opts...))

	err := s.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid planner options: %w", err)
	}

	tf, err := tfexec.NewTerraform(s.workdir, s.terraformBin)
	if err != nil {
		return nil, fmt.Errorf("failed to create Terraform executor: %w", err)
	}

	return &Planner{
		tf:          tf,
		skipInit:    s.skipInit,
		skipRefresh: s.skipRefresh,
	}, nil
}

// Plan runs Terraform commands to obtain a plan containing all resources
// Terraform plans to create and delete in this module, including all of those
// resources' known attributes.
func (p *Planner) Plan(ctx context.Context) (terraform.Plan, error) {
	if !p.skipInit {
		logger.Info("Running \"terraform init\"...")

		err := p.tf.Init(ctx)
		if err != nil {
			return terraform.Plan{}, fmt.Errorf("failed to initialize Terraform: %w", err)
		}
	}

	if !p.skipRefresh {
		logger.Info("Running \"terraform refresh\"...")

		err := p.tf.Refresh(ctx)
		if err != nil {
			return terraform.Plan{}, fmt.Errorf("failed to refresh Terraform state: %w", err)
		}
	}

	logger.Info("Running \"terraform plan\"...")

	planFile, err := os.CreateTemp("", "tfautomv.*.plan")
	if err != nil {
		return terraform.Plan{}, fmt.Errorf("failed to create temporary file to store raw plan: %w", err)
	}
	defer os.Remove(planFile.Name())

	_, err = p.tf.Plan(ctx, tfexec.Out(planFile.Name()))
	if err != nil {
		return terraform.Plan{}, fmt.Errorf("failed to compute Terraform plan: %w", err)
	}

	planOutput, err := p.tf.ShowPlanFile(ctx, planFile.Name())
	if err != nil {
		return terraform.Plan{}, fmt.Errorf("failed to read raw Terraform plan: %w", err)
	}

	plan, err := p.summarize(planOutput)
	if err != nil {
		return terraform.Plan{}, fmt.Errorf("failed to summarize Terraform plan: %w", err)
	}

	return plan, nil
}

func (p *Planner) summarize(hashicorpDTO *tfjson.Plan) (terraform.Plan, error) {
	plan := terraform.Plan{
		Module: terraform.Module{
			Path: p.tf.WorkingDir(),
		},
	}

	for _, rc := range hashicorpDTO.ResourceChanges {
		isCreated := slices.Contains(rc.Change.Actions, tfjson.ActionCreate)
		isDestroyed := slices.Contains(rc.Change.Actions, tfjson.ActionDelete)

		if !isCreated && !isDestroyed {
			continue
		}

		if isCreated {
			attributes, err := flatmap.Flatten(rc.Change.After)
			if err != nil {
				return terraform.Plan{}, fmt.Errorf("failed to flatten attributes of %s: %w", rc.Address, err)
			}

			r := terraform.Resource{
				Module:     plan.Module,
				Type:       rc.Type,
				Address:    rc.Address,
				Attributes: attributes,
			}

			plan.PlannedForCreation = append(plan.PlannedForCreation, r)
		}

		if isDestroyed {
			attributes, err := flatmap.Flatten(rc.Change.Before)
			if err != nil {
				return terraform.Plan{}, fmt.Errorf("failed to flatten attributes of %s: %w", rc.Address, err)
			}

			r := terraform.Resource{
				Module:     plan.Module,
				Type:       rc.Type,
				Address:    rc.Address,
				Attributes: attributes,
			}

			plan.PlannedForDeletion = append(plan.PlannedForDeletion, r)
		}
	}

	return plan, nil
}
