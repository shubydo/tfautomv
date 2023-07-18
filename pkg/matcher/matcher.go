package matcher

// Matcher finds pairs of Terraform resources that are likely the same resource.
type Matcher struct {
	differ Differ
}

// A Differ compares Terraform resources.
type Differ interface {
	// Diff compares a resource Terraform plans to create to a resource
	// Terraform plans to delete.
	Diff(create, delete terraform.Resource) terraform.Diff
}

func defaultOptions() []Option {
	return nil
}

// New builds a new Matcher configured with the provided options.
func New(opts ...Option) (*Matcher, error) {
	s := newSettings()

	s.apply(append(defaultOptions(), opts...))

	err := s.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid matcher options: %w", err)
	}

	return &Matcher{
		differ: s.differ
	}, nil
}

// FindMatches compares each resource Terraform plans to create to each resource
// Terraform plans to delete of the same type. If there is no difference between
// the two resources, then those resources match.
func (m *Matcher) FindMatches(plans ...terraform.Plan) []terraform.Match {
	
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

	// Then, compare resources of the same type to find all matches.
	var matches []terraform.Match
	for t := range createByType {
		for _, c := range createByType[t] {
			for _, d := range deleteByType[t] {
				diff := m.differ.Diff(c, d)
				if diff.IsMatch() {
					matches = append(matches, terraform.NewMatch(c, d))
				}
			}
		}
	}

	return matches
}
