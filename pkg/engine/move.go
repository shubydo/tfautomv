package engine

import "sort"

// A Move represents a Terraform resource that we should move from one address
// to another. A resource can be moved within the same module or to a different
// module.
type Move struct {
	// The module the resource is being moved from.
	SourceModule string
	// The module the resource is being moved to. This is equal to SourceModule
	// when the resource is being moved within the same module.
	DestinationModule string

	// The resource's address before the move.
	SourceAddress string
	// The resource's address after the move.
	DestinationAddress string
}

func DetermineMoves(comparisons []ResourceComparison) []Move {

	// We choose to move a resource planned for deletion to a resource planned
	// for creation if and only if the resources match each other and
	// only each other.

	matchCountResourcePlannedForCreation := make(map[string]int)
	matchCountResourcePlannedForDeletion := make(map[string]int)
	for _, comparison := range comparisons {
		if comparison.IsMatch() {
			matchCountResourcePlannedForCreation[comparison.PlannedForCreation.ID()]++
			matchCountResourcePlannedForDeletion[comparison.PlannedForDeletion.ID()]++
		}
	}

	var moves []Move

	for _, comparison := range comparisons {
		if !comparison.IsMatch() {
			continue
		}

		if matchCountResourcePlannedForCreation[comparison.PlannedForCreation.ID()] != 1 {
			continue
		}

		if matchCountResourcePlannedForDeletion[comparison.PlannedForDeletion.ID()] != 1 {
			continue
		}

		m := Move{
			SourceModule:       comparison.PlannedForDeletion.ModuleID,
			SourceAddress:      comparison.PlannedForDeletion.Address,
			DestinationModule:  comparison.PlannedForCreation.ModuleID,
			DestinationAddress: comparison.PlannedForCreation.Address,
		}
		moves = append(moves, m)
	}

	// Sort the moves so that the result is deterministic.
	sortMoves(moves)

	return moves
}

func sortMoves(moves []Move) {
	sort.Slice(moves, func(i, j int) bool {
		a, b := moves[i], moves[j]

		switch {
		case a.SourceModule != b.SourceModule:
			return a.SourceModule < b.SourceModule

		case a.DestinationModule != b.DestinationModule:
			return a.DestinationModule < b.DestinationModule

		case a.SourceAddress != b.SourceAddress:
			return a.SourceAddress < b.SourceAddress

		case a.DestinationAddress != b.DestinationAddress:
			return a.DestinationAddress < b.DestinationAddress

		default:
			return false // a == b so it doesn't matter what we return here
		}
	})
}
