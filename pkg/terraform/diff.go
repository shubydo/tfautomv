package terraform

// A Diff represents the differences between a resource Terraform plans to
// create and a resource Terraform plans to delete.
type Diff struct {
	// Keys of attributes that have the same value in both resources.
	MatchingAttributes []string

	// Keys of attributes that have different values in both resources.
	// Attributes absent from the resource Terraform plans to create are not
	// included because their values are only known after the resource has been
	// created.
	MismatchingAttributes []string

	// Keys of attributes that would normally be mismatching, but where the user
	// provided a rule that says to ignore that particular difference.
	IgnoredAttributes []string
}

func (d Diff) IsMatch() bool {
	return len(d.MismatchingAttributes) == 0
}