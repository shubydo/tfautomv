package differ

import (
	"fmt"
	"sort"

	"github.com/busser/tfautomv/pkg/differ/rules"
	"github.com/busser/tfautomv/pkg/terraform"
)

// Differ compares a resource Terraform plans to create with a resource
// Terraform plans to delete.
type Differ struct {
	rules []rules.Rule
}

func defaultOptions() []Option {
	return nil
}

// New builds a new Differ configured with the provided options.
func New(opts ...Option) (*Differ, error) {
	s := newSettings()

	s.apply(append(defaultOptions(), opts...))

	err := s.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid differ options: %w", err)
	}

	return &Differ{
		rules: s.rules,
	}, nil
}

// Diff compares two Terraform resources' attributes and returns information
// about what attributes have different values. The first resource must be one
// Terraform plans to create, and the second resource must be one Terraform
// plans to delete.
//
// If the two resources are not of the same type, Diff will panic.
//
// Diff ignores attributes that are only in the resource planned for deletion,
// because that means those attributes' values are only known after the resource
// has been created.
func (d *Differ) Diff(create, delete terraform.Resource) terraform.Diff {
	var matching, mismatching, ignored []string

	if create.Type != delete.Type {
		panic(fmt.Sprintf("resources are of different types: %s and %s", create.Type, delete.Type))
	}

	for key, cValue := range create.Attributes {
		if cValue == nil {
			continue
		}

		dValue, isSet := delete.Attributes[key]

		if isSet && cValue == dValue {
			// Both values are identical: it's a match.
			matching = append(matching, key)
			continue
		}

		var ruleSaysToIgnore bool
		for _, r := range d.rules {
			if !r.AppliesTo(create.Type, key) {
				continue
			}

			if r.Equates(cValue, dValue) {
				ruleSaysToIgnore = true
				break
			}
		}

		if ruleSaysToIgnore {
			// A rule says to ignore the difference between the two values.
			ignored = append(ignored, key)
			continue
		}

		// The two values are different and no rule says to ignore the difference.
		mismatching = append(mismatching, key)
	}

	// We sort the keys so that the final diff is deterministic.
	sort.Strings(matching)
	sort.Strings(mismatching)
	sort.Strings(ignored)

	return terraform.Diff{
		MatchingAttributes:    matching,
		MismatchingAttributes: mismatching,
		IgnoredAttributes:     ignored,
	}
}
