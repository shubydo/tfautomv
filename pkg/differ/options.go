package differ

import "github.com/busser/tfautomv/pkg/differ/rules"

// An Option configures a Differ's behavior.
type Option func(*settings)

// WithRules adds rules to the Differ.
func WithRules(rules ...rules.Rule) Option {
	return func(s *settings) {
		s.rules = append(s.rules, rules...)
	}
}

type settings struct {
	rules []rules.Rule
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
	return nil
}
