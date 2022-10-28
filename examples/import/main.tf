terraform {
  required_providers {
    hashicups = {
      version = "0.2"
      source  = "petetanton.com/namespace/rundeck"
    }
  }
}

provider "hashicups" {
  username = "education"
  password = "test123"
}

resource "hashicups_order" "sample" {}

output "sample_order" {
  value = hashicups_order.sample
}
