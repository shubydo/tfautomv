package terraform

import "sort"

// A Move represents a Terraform state mv operation, where a resource is moved
// from one address to another. A resource can be moved within the same module
// or to a different module.
type Move struct {
	// The module the resource is being moved from.
	SourceModule Module
	// The module the resource is being moved to. This is equal to FromModule
	// if the resource is being moved within the same module.
	DestinationModule Module

	// The resource's address before the move.
	SourceAddress string
	// The resource's address after the move.
	DestinationAddress string
}

// NewMove builds a Move from two Resources.
func NewMove(source, destination Resource) Move {
	return Move{
		SourceModule:       source.Module,
		DestinationModule:  destination.Module,
		SourceAddress:      source.Address,
		DestinationAddress: destination.Address,
	}
}

// SortMoves puts the given moves in an arbitrary, deterministic order.
func SortMoves(moves []Move) {
	sort.Slice(moves, func(i, j int) bool {
		a, b := moves[i], moves[j]

		// Sort by module before address and, within the same module, by source
		// before target.

		switch {
		case a.SourceModule.Path != b.SourceModule.Path:
			return a.SourceModule.Path < b.SourceModule.Path
		case a.DestinationModule.Path != b.DestinationModule.Path:
			return a.DestinationModule.Path < b.DestinationModule.Path
		case a.SourceAddress != b.SourceAddress:
			return a.SourceAddress < b.SourceAddress
		case a.DestinationAddress != b.DestinationAddress:
			return a.DestinationAddress < b.DestinationAddress
		default:
			return false // a == b so it doesn't matter what we return here
		}
	})
}
