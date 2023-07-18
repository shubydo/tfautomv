package writer

import (
	"fmt"
	"io"
	"os"

	"github.com/busser/tfautomv/pkg/terraform"
)

type Writer struct {
	format Format
}

type Format string

const (
	FormatBlocks   Format = "blocks"
	FormatCommands Format = "commands"
)

func defaultOptions() []Option {
	return []Option{
		WithFormat(FormatBlocks),
	}
}

func New(opts ...Option) (*Writer, error) {
	s := newSettings()

	s.apply(append(defaultOptions(), opts...))

	err := s.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid generator options: %w", err)
	}

	return &Writer{
		format: s.format,
	}, nil
}

func (g *Writer) Write(moves []terraform.Move) error {
	switch g.format {
	case FormatBlocks:
		// TODO: Handle different modules
		return appendToFile(moves, "moves.tf")
	case FormatCommands:
		return writeShellCommands(moves, os.Stdout)
	}

	return nil
}

func appendToFile(moves []terraform.Move, path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, m := range moves {
		fmt.Fprintf(f, "moved {\n  from = %s\n  to   = %s\n}\n", m.SourceAddress, m.DestinationAddress)
	}

	return nil
}

func writeShellCommands(moves []terraform.Move, w io.Writer) error {
	// Start with moves within the same module.
	for _, m := range moves {
		if m.SourceModule == m.DestinationModule {
			var chdirFlag string
			if m.SourceModule.Path != "." {
				chdirFlag = fmt.Sprintf("-chdir %q ", m.SourceModule.Path)
			}
			_, err := fmt.Fprintf(w, "terraform %s state mv %q %q\n",
				chdirFlag, m.SourceAddress, m.DestinationAddress)
			if err != nil {
				return err
			}
		}
	}

	// TODO: Handle moves between modules.

	return nil
}
