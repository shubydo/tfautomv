package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	flag "github.com/spf13/pflag"

	"github.com/busser/tfautomv/pkg/engine"
	"github.com/busser/tfautomv/pkg/engine/rules"
	"github.com/busser/tfautomv/pkg/pretty"
	"github.com/busser/tfautomv/pkg/terraform"
)

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString(pretty.Error(err) + "\n")
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

	planOptions := []terraform.PlanOption{
		terraform.WithTerraformBin(terraformBin),
		terraform.WithSkipInit(skipInit),
		terraform.WithSkipRefresh(skipRefresh),
	}

	// TODO: suppress output when --quiet is set.
	plans, err := getPlans(context.TODO(), modulePaths, planOptions)
	if err != nil {
		return err
	}

	var userRules []engine.Rule
	for _, raw := range ignoreRules {
		rule, err := rules.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid rule passed with -ignore flag %q: %w", raw, err)
		}

		userRules = append(userRules, rule)
	}

	comparisons := engine.CompareAll(engine.MergePlans(plans), userRules)
	moves := engine.DetermineMoves(comparisons)

	if !quiet {
		summarizer := pretty.NewSummarizer(moves, comparisons, verbosity)
		summary := summarizer.Summary()
		os.Stderr.WriteString("\n" + summary + "\n\n")
	}

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

		if !quiet {
			os.Stderr.WriteString(pretty.Colorf("%s written to [bold][green]%s", pretty.StyledNumMoves(len(moves)), movesFilePath))
			os.Stderr.WriteString("\n")
		}
	case "commands":
		err := terraform.WriteMoveCommands(os.Stdout, terraformMoves)
		if err != nil {
			return fmt.Errorf("failed to write move commands: %w", err)
		}

		if !quiet {
			os.Stderr.WriteString(pretty.Colorf("%s written to [bold][green]standard output", pretty.StyledNumMoves(len(moves))))
			os.Stderr.WriteString("\n")
		}
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
	quiet        bool
	skipInit     bool
	skipRefresh  bool
	terraformBin string
	verbosity    int
)

func parseFlags() {
	flag.StringSliceVar(&ignoreRules, "ignore", nil, "ignore differences based on a `rule`")
	flag.BoolVar(&noColor, "no-color", false, "disable color in output")
	flag.StringVarP(&outputFormat, "output", "o", "blocks", "output `format` of moves (\"blocks\" or \"commands\")")
	flag.BoolVarP(&printVersion, "version", "V", false, "print version and exit")
	flag.BoolVarP(&quiet, "quiet", "q", false, "suppress all human-readable output")
	flag.BoolVarP(&skipInit, "skip-init", "s", false, "skip running terraform init")
	flag.BoolVarP(&skipRefresh, "skip-refresh", "S", false, "skip running terraform refresh")
	flag.StringVar(&terraformBin, "terraform-bin", "terraform", "terraform binary to use")
	flag.CountVarP(&verbosity, "verbose", "v", "increase verbosity (can be specified multiple times)")

	flag.Parse()
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

func getPlans(ctx context.Context, workdirs []string, options []terraform.PlanOption) ([]engine.Plan, error) {
	type result struct {
		plan engine.Plan
		err  error
	}
	results := make([]result, len(workdirs))

	getPlan := func(i int) {
		jsonPlan, err := terraform.GetPlan(ctx, workdirs[i], options...)
		if err != nil {
			results[i].err = fmt.Errorf("failed to get plan for workdir %q: %w", workdirs[i], err)
			return
		}

		plan, err := engine.SummarizeJSONPlan(workdirs[i], jsonPlan)
		if err != nil {
			results[i].err = fmt.Errorf("failed to summarize plan for workdir %q: %w", workdirs[i], err)
			return
		}

		results[i].plan = plan
	}

	var wg sync.WaitGroup
	for i := range workdirs {
		wg.Add(1)
		go func(i int) {
			getPlan(i)
			wg.Done()
		}(i)
	}

	wg.Wait()

	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	var plans []engine.Plan
	for _, r := range results {
		plans = append(plans, r.plan)
	}

	return plans, nil
}
