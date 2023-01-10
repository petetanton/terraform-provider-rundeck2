package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"terraform-provider-rundeck/rundeck"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderAddr: "registry.terraform.io/petetanton/rundeck",
		ProviderFunc: func() *schema.Provider {
			return rundeck.Provider()
		},
	})
}
