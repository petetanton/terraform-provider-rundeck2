terraform {
  required_providers {
    rundeck = {
      version = "0.2"
      source  = "petetanton.com/namespace/rundeck"
    }
  }
}

provider "rundeck" {
  url         = "https://rundeck.plat.dazn-dev.com"
  api_version = "26"
  auth_token  = ""
}

data "rundeck_project" "sre" {
  name = "sre"
}

# resource "rundeck_project" "test-project" {
#   name        = "TestProject"
#   description = "Nice one"
#   resource_model_source {
#     config = {
#       "file"        = "/home/rundeck/node-provider.py"
#       "format"      = "resourcejson"
#       "interpreter" = "python3"
#     }
#     type = "script"
#   }

#   resource_model_source {
#     config = {
#       "format"      = "local"
#     }
#     type = "local"
#   }
# }

output "sre" {
  value = data.rundeck_project.sre
}

# output "test" {
#   value = rundeck_project.test-project
# }
