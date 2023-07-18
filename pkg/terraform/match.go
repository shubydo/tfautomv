package terraform

// A Match represents a pair of Terraform resources: one that Terraform plans to
// create and another that Terraform plans to delete. These resources are said
// to match when there is seemingly no difference between the two resources.
type Match struct {
	// The resource Terraform plans to create.
	PlannedForCreation Resource

	// The resource Terraform plans to delete.
	PlannedForDeletion Resource
}
