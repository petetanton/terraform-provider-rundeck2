package rundeck

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rundeck/go-rundeck/rundeck"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_URL", nil),
				Description: "URL of the root of the target Rundeck server.",
			},
			"api_version": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_API_VERSION", "14"),
				Description: "API Version of the target Rundeck server.",
			},
			"auth_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("RUNDECK_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Rundeck API.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"rundeck_project": resourceRundeckProject(),
			"rundeck_acl":     resourceRundeckAclPolicy(),
			"hashicups_order": resourceOrder(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"hashicups_coffees": dataSourceCoffees(),
			"hashicups_order":   dataSourceOrder(),
			"rundeck_project":   dataSourcesProject(),
			"rundeck_acl":       dataSourcesAcl(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	urlP, _ := d.Get("url").(string)
	apiVersion, _ := d.Get("api_version").(string)
	token := d.Get("auth_token").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	apiURLString := fmt.Sprintf("%s/api/%s", urlP, apiVersion)
	apiURL, err := url.Parse(apiURLString)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to parse the API URL",
			Detail:   fmt.Sprintf("Unable to parse: '%s' as a valid URL.", apiURLString),
		})
		return nil, diags
	}

	if token == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "No token provider to access Rundeck",
			Detail:   "Rundeck requires token based authentication.",
		})
		return nil, diags
	}

	client := rundeck.NewRundeckWithBaseURI(apiURL.String())
	client.Authorizer = &auth.TokenAuthorizer{Token: token}

	return &client, diags
}
