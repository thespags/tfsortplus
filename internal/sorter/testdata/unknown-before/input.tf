resource "gitlab_project" "test" {
  name = "test"
}

resource "aws_instance" "example" {
  ami = "ami-123"
}
