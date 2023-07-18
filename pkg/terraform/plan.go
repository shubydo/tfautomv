package terraform

// TODO
type Plan struct {
	Module             Module
	PlannedForCreation []Resource
	PlannedForDeletion []Resource
}
