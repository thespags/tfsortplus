data "aws_vpc" "main" {
  default = true # Use default VPC
}

resource "aws_instance" "example" {
  ami = "ami-123" # AMI ID
  instance_type = "t2.micro"
}
