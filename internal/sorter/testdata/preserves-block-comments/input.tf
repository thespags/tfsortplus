# This is a header comment

# Resource comment
resource "gitlab_project" "test" {
  name = "test"
}

# Data source comment
# with multiple lines
data "gitlab_group" "parent" {
  full_path = "example"
}
