resource "aws_instance" "example" {
  ami = "ami-123"
}

output "name" {
  value = "test"
}
