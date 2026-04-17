# This is a header comment
# with multiple lines

resource "gitlab_branch_protection" "main" {
  branch = "main"
}

resource "gitlab_project" "test" {
  name = "test"
}
