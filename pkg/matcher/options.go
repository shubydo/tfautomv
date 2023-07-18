package matcher

import "fmt"

// An Option configures a Matcher's behavior.
type Option func(*settings)

// WithDiffer sets the Differ the Matcher uses.
func WithDiffer(d Differ) Option {
	return func(s *settings) {
		s.differ = d
	}
}

type settings struct {
	differ Differ
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
	if s.differ == nil {
		return fmt.Errorf("no differ provided")
	}
	return nil
}
