package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/busser/tfautomv/pkg/differ"
	"github.com/busser/tfautomv/pkg/differ/rules"
	"github.com/busser/tfautomv/pkg/matcher"
	"github.com/busser/tfautomv/pkg/mover"
	"github.com/busser/tfautomv/pkg/planner"
	"github.com/busser/tfautomv/pkg/terraform"
	"github.com/busser/tfautomv/pkg/writer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

//go:embed VERSION
var tfautomvVersion string

func run() error {
	parseFlags()

	if printVersion {
		fmt.Println(tfautomvVersion)
		return nil
	}

	modulePaths := flag.Args()
	if len(modulePaths) == 0 {
		modulePaths = []string{"."}
	}

	ctx := context.TODO()

	var plans []terraform.Plan
	for _, modulePath := range modulePaths {
		planner, err := planner.New(
			planner.WithWorkdir(modulePath),
			planner.WithTerraformBin(terraformBin),
		)
		if err != nil {
			return fmt.Errorf("failed to create planner for module %q: %w", modulePath, err)
		}
		plan, err := planner.Plan(ctx)
		if err != nil {
			return fmt.Errorf("failed to plan module %q: %w", modulePath, err)
		}
		plans = append(plans, plan)
	}

	var userRules []rules.Rule
	for _, raw := range ignoreRules {
		r, err := rules.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid rule passed with -ignore flag %q: %w", raw, err)
		}
		userRules = append(userRules, r)
	}

	differ, err := differ.New(differ.WithRules(userRules...))
	if err != nil {
		return fmt.Errorf("failed to create differ: %w", err)
	}

	matcher, err := matcher.New(matcher.WithDiffer(differ))
	if err != nil {
		return fmt.Errorf("failed to create matcher: %w", err)
	}

	matches := matcher.FindMatches(plans...)

	moves := mover.New().FindMoves(matches)

	writer, err := writer.New(writer.WithFormat(writer.Format(outputFormat)))
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	err = writer.Write(moves)
	if err != nil {
		return fmt.Errorf("failed to write moves: %w", err)
	}

	return nil
}

// Flags
var (
	dryRun       bool
	ignoreRules  []string
	noColor      bool
	outputFormat string
	printVersion bool
	showAnalysis bool
	terraformBin string
)

func parseFlags() {
	flag.BoolVar(&dryRun, "dry-run", false, "print moves instead of writing them to disk")
	flag.Var(stringSliceValue{&ignoreRules}, "ignore", "ignore differences based on a `rule`")
	flag.BoolVar(&noColor, "no-color", false, "disable color in output")
	flag.StringVar(&outputFormat, "output", "blocks", "output `format` of moves (\"blocks\" or \"commands\")")
	flag.BoolVar(&showAnalysis, "show-analysis", false, "show detailed analysis of Terraform plan")
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
