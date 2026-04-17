module "vpc" {
  source = "./modules/vpc"
}

resource "gitlab_project" "test" {
  name = "test"
}

module "group" {
  source = "./modules/group"
}
