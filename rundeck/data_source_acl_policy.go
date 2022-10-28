package rundeck

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rundeck/go-rundeck/rundeck"
)

func dataSourcesAcl() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAclRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name for the ACL policy",
				ForceNew:    true,
			},
			"policy": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "YAML formatted ACL Policy string",
				ForceNew:    false,
			},
		},
	}
}

func dataSourceAclRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Get("name").(string)

	acl, err := client.SystemACLPolicyGet(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if acl.StatusCode == 404 {
		return diag.FromErr(fmt.Errorf("acl not found: (%s)", name))
	}

	d.Set("name", name)
	d.Set("policy", acl.Contents)

	return diags
}
