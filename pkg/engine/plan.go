package engine

import (
	"fmt"
	"sort"

	"github.com/busser/tfautomv/pkg/engine/flatmap"
	"github.com/busser/tfautomv/pkg/slices"
	tfjson "github.com/hashicorp/terraform-json"
)

// TODO: Document.
type Plan struct {
	ModuleID           string
	PlannedForCreation []Resource
	PlannedForDeletion []Resource
}

// TODO: Document.
func SummarizeJSONPlan(moduleID string, jsonPlan *tfjson.Plan) (Plan, error) {
	plan := Plan{
		ModuleID: moduleID,
	}

	for _, rc := range jsonPlan.ResourceChanges {
		isCreated := slices.Contains(rc.Change.Actions, tfjson.ActionCreate)
		isDestroyed := slices.Contains(rc.Change.Actions, tfjson.ActionDelete)

		if !isCreated && !isDestroyed {
			continue
		}

		if isCreated {
			attributes, err := flatmap.Flatten(rc.Change.After)
			if err != nil {
				return Plan{}, fmt.Errorf("failed to flatten attributes of %s: %w", rc.Address, err)
			}

			r := Resource{
				ModuleID:   moduleID,
				Type:       rc.Type,
				Address:    rc.Address,
				Attributes: attributes,
			}

			plan.PlannedForCreation = append(plan.PlannedForCreation, r)
		}

		if isDestroyed {
			attributes, err := flatmap.Flatten(rc.Change.Before)
			if err != nil {
				return Plan{}, fmt.Errorf("failed to flatten attributes of %s: %w", rc.Address, err)
			}

			r := Resource{
				ModuleID:   moduleID,
				Type:       rc.Type,
				Address:    rc.Address,
				Attributes: attributes,
			}

			plan.PlannedForDeletion = append(plan.PlannedForDeletion, r)
		}
	}

	return plan, nil
}

// TODO: Document.
func ComparePlans(plans []Plan, rules []Rule) []ResourceComparison {
	// First, group resources by type and the action Terraform plans to take.
	createByType := make(map[string][]Resource)
	deleteByType := make(map[string][]Resource)
	for _, p := range plans {
		for _, r := range p.PlannedForCreation {
			createByType[r.Type] = append(createByType[r.Type], r)
		}
		for _, r := range p.PlannedForDeletion {
			deleteByType[r.Type] = append(deleteByType[r.Type], r)
		}
	}

	// Then, compare each resource Terraform plans to create to all resources
	// Terraform plans to delete of the same type.
	var comparisons []ResourceComparison
	for t := range createByType {
		for _, c := range createByType[t] {
			for _, d := range deleteByType[t] {
				comparison := CompareResources(c, d, rules)
				comparisons = append(comparisons, comparison)
			}
		}
	}

	// Finally, sort the comparisons so that the result is deterministic.
	sortComparisons(comparisons)

	return comparisons
}

func sortComparisons(comparisons []ResourceComparison) {
	sort.Slice(comparisons, func(i, j int) bool {
		a, b := comparisons[i], comparisons[j]

		// The goal here is to group comparisons that are for the same resource
		// together.

		switch {
		case a.PlannedForCreation.ModuleID != b.PlannedForCreation.ModuleID:
			return a.PlannedForCreation.ModuleID < b.PlannedForCreation.ModuleID

		case a.PlannedForCreation.Address != b.PlannedForCreation.Address:
			return a.PlannedForCreation.Address < b.PlannedForCreation.Address

		case a.PlannedForDeletion.ModuleID != b.PlannedForDeletion.ModuleID:
			return a.PlannedForDeletion.ModuleID < b.PlannedForDeletion.ModuleID

		case a.PlannedForDeletion.Address != b.PlannedForDeletion.Address:
			return a.PlannedForDeletion.Address < b.PlannedForDeletion.Address

		default:
			return false // a == b so it doesn't matter what we return here
		}
	})
}
