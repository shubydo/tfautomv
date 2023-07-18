package writer

import (
	"fmt"

	"github.com/busser/tfautomv/pkg/slices"
)

// An Option configures a Writer's behavior.
type Option func(*settings)

// WithFormat sets the format the Writer uses.
func WithFormat(f Format) Option {
	return func(s *settings) {
		s.format = f
	}
}

type settings struct {
	format Format
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
	validFormats := []Format{
		FormatBlocks,
		FormatCommands,
	}
	if !slices.Contains(validFormats, s.format) {
		return fmt.Errorf("unknown format %q", s.format)
	}

	return nil
}
