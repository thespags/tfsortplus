resource "gitlab_project" "test" {
  name = "test"
}

resource "gitlab_branch_protection" "main" {
  branch = "main"
}
