resource "aws_instance" "example" {
  ami = "ami-123"
}

locals {
  name = "test"
}
