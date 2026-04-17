resource "aws_instance" "example" {
  ami = "ami-123"
}

resource "gitlab_project" "test" {
  name = "test"
}

data "gitlab_group" "parent" {
  full_path = "example"
}
