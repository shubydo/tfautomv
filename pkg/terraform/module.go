package terraform

// A Module represents a Terraform root module.
type Module struct {
	// The Path to a directory containing the module. This is the directory
	// where we run Terraform commands like `terraform plan` or
	// `terraform state mv`. This is a unique identifier for the module.
	Path string
}
