package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/busser/tfautomv/pkg/engine"
	"github.com/busser/tfautomv/pkg/engine/rules"
	"github.com/busser/tfautomv/pkg/pretty"
	"github.com/busser/tfautomv/pkg/terraform"
)

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString(pretty.Error(err))
		os.Exit(1)
	}
}

//go:embed VERSION
var tfautomvVersion string

func run() error {
	parseFlags()

	if noColor {
		pretty.DisableColors()
	}

	if printVersion {
		fmt.Println(tfautomvVersion)
		return nil
	}

	err := smokeTests()
	if err != nil {
		return err
	}

	modulePaths := flag.Args()
	if len(modulePaths) == 0 {
		modulePaths = []string{"."}
	}

	ctx := context.TODO()

	planOptions := []terraform.PlanOption{
		terraform.WithTerraformBin(terraformBin),
	}

	var plans []engine.Plan
	for _, modulePath := range modulePaths {
		jsonPlan, err := terraform.GetPlan(ctx, modulePath, planOptions...)
		if err != nil {
			return fmt.Errorf("failed to get plan for module %q: %w", modulePath, err)
		}

		plan, err := engine.SummarizeJSONPlan(modulePath, jsonPlan)
		if err != nil {
			return fmt.Errorf("failed to summarize plan for module %q: %w", modulePath, err)
		}

		plans = append(plans, plan)
	}

	var userRules []engine.Rule
	for _, raw := range ignoreRules {
		rule, err := rules.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid rule passed with -ignore flag %q: %w", raw, err)
		}

		userRules = append(userRules, rule)
	}

	comparisons := engine.ComparePlans(plans, userRules)
	moves := engine.DetermineMoves(comparisons)

	summarizer := pretty.NewSummarizer(moves, comparisons)
	summary := summarizer.Summary(explain)
	os.Stderr.WriteString("\n" + summary + "\n\n")

	if len(moves) == 0 {
		return nil
	}

	terraformMoves := engineMovesToTerraformMoves(moves)

	switch outputFormat {
	case "blocks":
		if len(modulePaths) > 1 {
			return fmt.Errorf("blocks output format is not supported for multiple modules")
		}

		movesFilePath := filepath.Join(modulePaths[0], "moves.tf")
		movesFile, err := os.OpenFile(movesFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", movesFilePath, err)
		}

		err = terraform.WriteMovedBlocks(movesFile, terraformMoves)
		if err != nil {
			return fmt.Errorf("failed to write moved blocks: %w", err)
		}

		os.Stderr.WriteString(pretty.Colorf("\n%s written to [bold][green]%s[reset].\n", pretty.StyledNumMoves(len(moves)), movesFilePath))
	case "commands":
		err := terraform.WriteMoveCommands(os.Stdout, terraformMoves)
		if err != nil {
			return fmt.Errorf("failed to write move commands: %w", err)
		}

		os.Stderr.WriteString(pretty.Colorf("\n%s written to [bold][green]standard output[reset].\n", pretty.StyledNumMoves(len(moves))))
	default:
		return fmt.Errorf("unknown output format %q", outputFormat)
	}

	return nil
}

// Flags
var (
	ignoreRules  []string
	noColor      bool
	outputFormat string
	printVersion bool
	explain      bool
	terraformBin string
)

func parseFlags() {
	flag.Var(stringSliceValue{&ignoreRules}, "ignore", "ignore differences based on a `rule`")
	flag.BoolVar(&noColor, "no-color", false, "disable color in output")
	flag.StringVar(&outputFormat, "output", "blocks", "output `format` of moves (\"blocks\" or \"commands\")")
	flag.BoolVar(&explain, "explain", false, "explain why resources are moved or not moved")
	flag.BoolVar(&printVersion, "version", false, "print version and exit")
	flag.StringVar(&terraformBin, "terraform-bin", "terraform", "terraform binary to use")

	flag.Parse()
}

type stringSliceValue struct {
	s *[]string
}

func (v stringSliceValue) String() string {
	if v.s == nil || *v.s == nil {
		return ""
	}
	return fmt.Sprintf("%q", *v.s)
}

func (v stringSliceValue) Set(raw string) error {
	*v.s = append(*v.s, raw)
	return nil
}

func engineMovesToTerraformMoves(moves []engine.Move) []terraform.Move {
	var terraformMoves []terraform.Move

	for _, m := range moves {
		terraformMoves = append(terraformMoves, terraform.Move{
			FromWorkdir: m.SourceModule,
			ToWorkdir:   m.DestinationModule,
			FromAddress: m.SourceAddress,
			ToAddress:   m.DestinationAddress,
		})
	}

	return terraformMoves
}

func smokeTests() error {
	if len(flag.Args()) > 1 && outputFormat == "blocks" {
		return fmt.Errorf("blocks output format is not supported for multiple modules")
	}
	return nil
}
