module "group" {
  source = "./modules/group"
}

resource "gitlab_project" "test" {
  name = "test"
}

module "vpc" {
  source = "./modules/vpc"
}
