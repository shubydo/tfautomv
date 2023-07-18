package writer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

	var commands []string

	// Start with moves within the same module.

	for _, m := range moves {
		if m.SourceModule == m.DestinationModule {
			var chdirFlag string
			if m.SourceModule.Path != "." {
				chdirFlag = fmt.Sprintf("-chdir=%q ", m.SourceModule.Path)
			}

			commands = append(commands,
				fmt.Sprintf("terraform %s state mv %q %q",
					chdirFlag, m.SourceAddress, m.DestinationAddress),
			)
		}
	}

	// Then, pull the states of all modules that require cross-module moves.

	var modules []string
	for _, m := range moves {
		if m.SourceModule != m.DestinationModule {
			modules = append(modules, m.SourceModule.Path, m.DestinationModule.Path)
		}
	}
	modules = unique(modules)
	sort.Strings(modules)

	const localCopyFileName = "tfautomv-local-copy.tfstate"
	backupFileName := fmt.Sprintf("tfautomv-backup-%d.tfstate", time.Now().Unix())

	for _, module := range modules {
		commands = append(commands,
			fmt.Sprintf("terraform -chdir=%q state pull > %q",
				module, filepath.Join(module, localCopyFileName)),
			fmt.Sprintf("cp %q %q",
				filepath.Join(module, localCopyFileName),
				filepath.Join(module, backupFileName)),
		)
	}

	// Next, perform all the moves.

	for _, move := range moves {
		if move.SourceModule == move.DestinationModule {
			// Already handled above.
			continue
		}

		commands = append(commands,
			fmt.Sprintf("terraform state mv -state=%q -state-out=%q %q %q",
				filepath.Join(move.SourceModule.Path, localCopyFileName),
				filepath.Join(move.DestinationModule.Path, localCopyFileName),
				move.SourceAddress,
				move.DestinationAddress),
		)
	}

	// Then, push the states of all modules we manipulated.

	for _, m := range modules {
		commands = append(commands,
			fmt.Sprintf("terraform -chdir=%q state push %q",
				m, localCopyFileName),
		)
	}

	// And we're done.

	_, err := fmt.Fprint(w, strings.Join(commands, "\n"))
	if err != nil {
		return err
	}

	return nil
}

func unique(s []string) []string {
	seen := make(map[string]struct{})
	var unique []string
	for _, e := range s {
		if _, ok := seen[e]; !ok {
			unique = append(unique, e)
			seen[e] = struct{}{}
		}
	}
	return unique
}
