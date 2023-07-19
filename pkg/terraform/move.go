package terraform

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TODO: Document.
type Move struct {
	FromWorkdir string
	ToWorkdir   string
	FromAddress string
	ToAddress   string
}

func (m Move) block() string {
	return fmt.Sprintf("moved {\n  from = %s\n  to   = %s\n}", m.FromAddress, m.ToAddress)
}

func (m Move) isWithinSameWorkdir() bool {
	return m.FromWorkdir == m.ToWorkdir
}

// TODO: Document.
func WriteMovedBlocks(w io.Writer, moves []Move) error {
	var blocks []string

	for _, move := range moves {
		if !move.isWithinSameWorkdir() {
			return fmt.Errorf("cannot write blocks for moves between different working directories")
		}

		blocks = append(blocks, move.block())
	}

	_, err := w.Write([]byte(strings.Join(blocks, "\n")))

	return err
}

// TODO: Document.
func WriteMoveCommands(w io.Writer, moves []Move) error {
	var commands []string

	// Start with moves within the same module.

	for _, m := range moves {
		if m.FromWorkdir == m.ToWorkdir {
			var chdirFlag string
			if m.FromWorkdir != "." {
				chdirFlag = fmt.Sprintf("-chdir=%q ", m.FromWorkdir)
			}

			commands = append(commands,
				fmt.Sprintf("terraform %s state mv %q %q",
					chdirFlag, m.FromAddress, m.ToAddress),
			)
		}
	}

	// Then, pull the states of all working directories that require
	// cross-directory moves.

	var workdirs []string
	for _, m := range moves {
		if m.FromWorkdir != m.ToWorkdir {
			workdirs = append(workdirs, m.FromWorkdir, m.ToWorkdir)
		}
	}
	workdirs = unique(workdirs)
	sort.Strings(workdirs)

	const localCopyFileName = "tfautomv-local-copy.tfstate"
	backupFileName := fmt.Sprintf("tfautomv-backup-%d.tfstate", time.Now().Unix())

	for _, module := range workdirs {
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
		if move.FromWorkdir == move.ToWorkdir {
			// Already handled above.
			continue
		}

		commands = append(commands,
			fmt.Sprintf("terraform state mv -state=%q -state-out=%q %q %q",
				filepath.Join(move.FromWorkdir, localCopyFileName),
				filepath.Join(move.ToWorkdir, localCopyFileName),
				move.FromAddress,
				move.ToAddress),
		)
	}

	// Then, push the states of all modules we manipulated.

	for _, m := range workdirs {
		commands = append(commands,
			fmt.Sprintf("terraform -chdir=%q state push %q",
				m, localCopyFileName),
		)
	}

	// And we're done.

	_, err := fmt.Fprint(w, strings.Join(commands, "\n"))

	return err
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
