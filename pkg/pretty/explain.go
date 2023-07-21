package pretty

import (
	"sort"
	"strings"

	"github.com/busser/tfautomv/pkg/engine"
)

// Explanation returns a human-readable explanation of the tfautomv engine's
// decisions to move or not move resources.
func Explanation(moves []engine.Move, comparisons []engine.ResourceComparison) string {
	e := newExplainer(moves, comparisons)

	return e.explanation()
}

type explainer struct {
	moves       []engine.Move
	comparisons []engine.ResourceComparison

	// used to explain moves
	modulesWithMoves []string

	// used to explain matches and non-matches
	resourcesToCreateByID  map[string]engine.Resource
	resourcesToDeleteByID  map[string]engine.Resource
	matchCountToCreateByID map[string]int
	matchCountToDeleteByID map[string]int
}

func newExplainer(moves []engine.Move, comparisons []engine.ResourceComparison) explainer {
	// We precompute some data to simplify the explanation logic.

	var modulesWithMoves []string
	for _, move := range moves {
		modulesWithMoves = append(modulesWithMoves, move.SourceModule, move.DestinationModule)
	}
	modulesWithMoves = unique(modulesWithMoves)
	sort.Strings(modulesWithMoves)

	resourcesToCreateByID := make(map[string]engine.Resource)
	resourcesToDeleteByID := make(map[string]engine.Resource)
	matchCountToCreateByID := make(map[string]int)
	matchCountToDeleteByID := make(map[string]int)
	for _, c := range comparisons {
		resourcesToCreateByID[c.PlannedForCreation.ID()] = c.PlannedForCreation
		resourcesToDeleteByID[c.PlannedForDeletion.ID()] = c.PlannedForDeletion
		if c.IsMatch() {
			matchCountToCreateByID[c.PlannedForCreation.ID()]++
			matchCountToDeleteByID[c.PlannedForDeletion.ID()]++
		}
	}

	return explainer{
		moves:       moves,
		comparisons: comparisons,

		modulesWithMoves: modulesWithMoves,

		resourcesToCreateByID: resourcesToCreateByID,
		resourcesToDeleteByID: resourcesToDeleteByID,

		matchCountToCreateByID: matchCountToCreateByID,
		matchCountToDeleteByID: matchCountToDeleteByID,
	}
}

func (e explainer) explanation() string {
	sections := []string{
		e.legend(),
	}

	sections = append(sections, e.moveExplanations()...)
	sections = append(sections, e.multipleMatchExplanations()...)
	sections = append(sections, e.noMatchExplanations()...)

	return strings.Join(sections, "\n\n")
}

func (e explainer) symbolCreate() string {
	return Color("[green][bold]+")
}

func (e explainer) symbolDelete() string {
	return Color("[red][bold]-")
}

func (e explainer) symbolIgnored() string {
	return Color("[yellow][bold]~")
}

func (e explainer) legend() string {
	lines := []string{
		"The following symbols are used in the explanation:",
		Colorf("  %s the resource Terraform plans to [green][bold]create[reset] has this attribute", e.symbolCreate()),
		Colorf("  %s the resource Terraform plans to [red][bold]delete[reset] has this attribute", e.symbolDelete()),
		Colorf("  %s differences in this attribute are [yellow][bold]ignored[reset] because of a rule", e.symbolIgnored()),
	}

	return strings.Join(lines, "\n")
}

func (e explainer) styledAddress(addr string) string {
	return Colorf("[bold]%s", addr)
}

func (e explainer) styledModule(module string) string {
	return Colorf("[bold]%s", module)
}

func (e explainer) annotationCreate() string {
	return Color("([green][bold]create[reset])")
}

func (e explainer) annotationDelete() string {
	return Color("([red][bold]delete[reset])")
}

func (e explainer) annotatedResource(r engine.Resource, annotation string) string {
	return Colorf("%s %s in %s", e.styledAddress(r.Address), annotation, e.styledModule(r.ModuleID))
}

func (e explainer) styledIgnored(comp engine.ResourceComparison) string {
	var lines []string

	for _, attr := range comp.IgnoredAttributes {
		lines = append(lines, Colorf("%s %s", e.symbolIgnored(), attr))
	}

	return strings.Join(lines, "\n")
}

func (e explainer) styledMismatches(comp engine.ResourceComparison) string {
	var lines []string

	for _, attr := range comp.MismatchingAttributes {
		lines = append(lines, Colorf("%s %s = %#v", e.symbolCreate(), attr, comp.PlannedForCreation.Attributes[attr]))
		lines = append(lines, Colorf("%s %s = %#v", e.symbolDelete(), attr, comp.PlannedForDeletion.Attributes[attr]))
	}

	return strings.Join(lines, "\n")
}

func (e explainer) styledMove(m engine.Move) string {
	comp := e.findComparison(m)

	var lines []string

	lines = append(lines, Colorf("from %s", e.styledAddress(comp.PlannedForDeletion.Address)))
	lines = append(lines, Colorf("to   %s", e.styledAddress(comp.PlannedForCreation.Address)))

	ignored := e.styledIgnored(comp)
	if ignored != "" {
		lines = append(lines, "")
		lines = append(lines, ignored)
	}

	return strings.Join(lines, "\n")
}

func (e explainer) styledNumMoves(n int) string {
	if n == 1 {
		return Color("[bold][green]1 move")
	}

	return Colorf("[bold][green]%d moves", n)
}

func (e explainer) styledMovesWithinModule(module string) string {
	var styledMoves []string
	for _, move := range e.moves {
		if move.SourceModule == module && move.DestinationModule == module {
			styledMoves = append(styledMoves, e.styledMove(move))
		}
	}

	if len(styledMoves) == 0 {
		return ""
	}

	header := Colorf("%s within %s", e.styledNumMoves(len(styledMoves)), e.styledModule(module))
	list := BoxItems("", styledMoves, "green")

	return header + "\n" + list
}

func (e explainer) styledMovesBetweenModules(fromModule, toModule string) string {
	var styledMoves []string
	for _, move := range e.moves {
		if move.SourceModule == fromModule && move.DestinationModule == toModule {
			styledMoves = append(styledMoves, e.styledMove(move))
		}
	}

	if len(styledMoves) == 0 {
		return ""
	}

	header := Colorf("%s from %s and %s", e.styledNumMoves(len(styledMoves)), e.styledModule(fromModule), e.styledModule(toModule))
	list := BoxItems("", styledMoves, "green")

	return header + "\n" + list
}

func (e explainer) moveExplanations() []string {
	var explanations []string

	for _, module := range e.modulesWithMoves {
		exp := e.styledMovesWithinModule(module)
		if exp != "" {
			explanations = append(explanations, exp)
		}
	}

	for _, fromModule := range e.modulesWithMoves {
		for _, toModule := range e.modulesWithMoves {
			if fromModule == toModule {
				continue
			}

			exp := e.styledMovesBetweenModules(fromModule, toModule)
			if exp != "" {
				explanations = append(explanations, exp)
			}
		}
	}

	return explanations
}

func (e explainer) styledAttributes(c engine.ResourceComparison) string {
	var lines []string

	ignored := e.styledIgnored(c)
	if ignored != "" {
		lines = append(lines, ignored)
	}

	mismatches := e.styledMismatches(c)
	if mismatches != "" {
		lines = append(lines, mismatches)
	}

	return strings.Join(lines, "\n")
}

func (e explainer) styledNumMatches(n int) string {
	if n == 0 {
		return Color("[bold][red]0 matches")
	}

	if n == 1 {
		return Color("[bold][magenta]1 match")
	}

	return Colorf("[bold][magenta]%d matches", n)
}

func (e explainer) styledMatchesForResourceToCreate(r engine.Resource) string {
	var styledMatches []string
	for _, c := range e.comparisons {
		if c.PlannedForCreation.ID() == r.ID() && c.IsMatch() {
			parts := []string{
				e.annotatedResource(c.PlannedForDeletion, e.annotationDelete()),
			}
			styledAttributes := e.styledAttributes(c)
			if styledAttributes != "" {
				parts = append(parts, "", styledAttributes)
			}

			styledMatches = append(styledMatches, strings.Join(parts, "\n"))
		}
	}

	header := Colorf("%s for %s", e.styledNumMatches(len(styledMatches)), e.annotatedResource(r, e.annotationCreate()))
	list := BoxItems("", styledMatches, "magenta")

	return header + "\n" + list
}

func (e explainer) styledMatchesForResourceToDelete(r engine.Resource) string {
	var styledMatches []string
	for _, c := range e.comparisons {
		if c.PlannedForDeletion.ID() == r.ID() && c.IsMatch() {
			parts := []string{
				e.annotatedResource(c.PlannedForCreation, e.annotationCreate()),
			}
			styledAttributes := e.styledAttributes(c)
			if styledAttributes != "" {
				parts = append(parts, "", styledAttributes)
			}

			styledMatches = append(styledMatches, strings.Join(parts, "\n"))
		}
	}

	header := Colorf("%s for %s", e.styledNumMatches(len(styledMatches)), e.annotatedResource(r, e.annotationDelete()))
	list := BoxItems("", styledMatches, "magenta")

	return header + "\n" + list
}

func (e explainer) multipleMatchExplanations() []string {
	var explanations []string

	for id, toCreate := range e.resourcesToCreateByID {
		if e.matchCountToCreateByID[id] > 1 {
			explanations = append(explanations, e.styledMatchesForResourceToCreate(toCreate))
		}
	}

	for id, toDelete := range e.resourcesToDeleteByID {
		if e.matchCountToDeleteByID[id] > 1 {
			explanations = append(explanations, e.styledMatchesForResourceToDelete(toDelete))
		}
	}

	return explanations
}

func (e explainer) styledNoMatchForResourceToCreate(r engine.Resource) string {
	var styledMatches []string
	for _, c := range e.comparisons {
		if c.PlannedForCreation.ID() == r.ID() && !c.IsMatch() && e.matchCountToDeleteByID[c.PlannedForDeletion.ID()] == 0 {
			parts := []string{
				e.annotatedResource(c.PlannedForDeletion, e.annotationDelete()),
			}
			styledAttributes := e.styledAttributes(c)
			if styledAttributes != "" {
				parts = append(parts, "", styledAttributes)
			}

			styledMatches = append(styledMatches, strings.Join(parts, "\n"))
		}
	}

	header := Colorf("%s for %s", e.styledNumMatches(0), e.annotatedResource(r, e.annotationCreate()))
	list := BoxItems("", styledMatches, "red")

	return header + "\n" + list
}

func (e explainer) styledNoMatchForResourceToDelete(r engine.Resource) string {
	var styledMatches []string
	for _, c := range e.comparisons {
		if c.PlannedForDeletion.ID() == r.ID() && !c.IsMatch() && e.matchCountToCreateByID[c.PlannedForCreation.ID()] == 0 {
			parts := []string{
				e.annotatedResource(c.PlannedForCreation, e.annotationCreate()),
			}
			styledAttributes := e.styledAttributes(c)
			if styledAttributes != "" {
				parts = append(parts, "", styledAttributes)
			}

			styledMatches = append(styledMatches, strings.Join(parts, "\n"))
		}
	}

	header := Colorf("%s for %s", e.styledNumMatches(0), e.annotatedResource(r, e.annotationDelete()))
	list := BoxItems("", styledMatches, "red")

	return header + "\n" + list
}

func (e explainer) noMatchExplanations() []string {
	var explanations []string

	for id, toCreate := range e.resourcesToCreateByID {
		if e.matchCountToCreateByID[id] == 0 {
			explanations = append(explanations, e.styledNoMatchForResourceToCreate(toCreate))
		}
	}

	for id, toDelete := range e.resourcesToDeleteByID {
		if e.matchCountToDeleteByID[id] == 0 {
			explanations = append(explanations, e.styledNoMatchForResourceToDelete(toDelete))
		}
	}

	return explanations
}

func (e explainer) findComparison(m engine.Move) engine.ResourceComparison {
	for _, c := range e.comparisons {
		if c.PlannedForCreation.ModuleID == m.DestinationModule &&
			c.PlannedForCreation.Address == m.DestinationAddress &&
			c.PlannedForDeletion.ModuleID == m.SourceModule &&
			c.PlannedForDeletion.Address == m.SourceAddress {
			return c
		}
	}

	return engine.ResourceComparison{}
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
