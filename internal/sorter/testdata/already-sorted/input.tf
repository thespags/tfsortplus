data "gitlab_group" "parent" {
  full_path = "example"
}

resource "gitlab_project" "test" {
  name = "test"
}
