package mover

import (
	"github.com/busser/tfautomv/pkg/terraform"
)

type Mover struct{}

func New() *Mover {
	return &Mover{}
}

func (m *Mover) FindMoves(matches []terraform.Match) []terraform.Move {

	// We choose to move a resource planned for destruction to a resource
	// planned for creation if and only if the resources match each other and
	// only each other.

	matchCountByResource := make(map[resourceKey]int)
	for _, match := range matches {
		matchCountByResource[makeResourceKey(match.PlannedForCreation)]++
		matchCountByResource[makeResourceKey(match.PlannedForDeletion)]++
	}

	var moves []terraform.Move

	for _, match := range matches {
		if matchCountByResource[makeResourceKey(match.PlannedForCreation)] != 1 {
			continue
		}

		if matchCountByResource[makeResourceKey(match.PlannedForDeletion)] != 1 {
			continue
		}

		m := terraform.Move{
			SourceModule:       match.PlannedForDeletion.Module,
			SourceAddress:      match.PlannedForDeletion.Address,
			DestinationModule:  match.PlannedForCreation.Module,
			DestinationAddress: match.PlannedForCreation.Address,
		}
		moves = append(moves, m)
	}

	return moves
}

type resourceKey struct {
	module  terraform.Module
	address string
}

func makeResourceKey(r terraform.Resource) resourceKey {
	return resourceKey{
		module:  r.Module,
		address: r.Address,
	}
}
