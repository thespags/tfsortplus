output "name" {
  value = "test"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}
