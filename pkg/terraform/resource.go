package terraform

// A Resource represents a Terraform resource.
type Resource struct {
	// The module where the resource is defined.
	Module Module

	// The resource's type.
	Type string

	// The resource's address within the module's state. Terraform uses the
	// address to map a module's source code to resources it manages.
	Address string

	// The resource's known attributes. In Terraform's state, attributes are
	// complex objects. We flatten them to make them easier to work with.
	//
	// For example, the following Terraform resource:
	//
	//	resource "aws_instance" "web" {
	//	  ami           = "ami-a1b2c3d4"
	//	  instance_type = "t2.micro"
	//
	//	  tags = {
	//	    Name        = "HelloWorld"
	//	    Environment = "Production"
	//	  }
	//	}
	//
	// Would be represented as the following Attributes:
	//
	//	Attributes{
	//	  "ami":              "ami-a1b2c3d4",
	//	  "instance_type":    "t2.micro",
	//	  "tags.Name":        "HelloWorld",
	//	  "tags.Environment": "Production",
	//	}
	//
	// Note that the "tags" attribute is flattened into "tags.Name" and
	// "tags.Environment".
	//
	// A value in the flattened map is either a string, a number, a boolean, or
	// nil. The nil value is used to represent null values in Terraform.
	Attributes map[string]any
}
