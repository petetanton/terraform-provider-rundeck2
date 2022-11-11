package rundeck

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckAclPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAclPolicyCreate,
		UpdateContext: resourceAclPolicyUpdate,
		ReadContext:   resourceAclPolicyRead,
		DeleteContext: resourceAclPolicyDelete,

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

func resourceAclPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	policy := d.Get("policy").(string)

	_, err := client.SystemACLPolicyCreate(ctx, name, &rundeck.SystemACLPolicyCreateRequest{
		Contents: &policy,
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call Client.SystemACLPolicyCreate"))
	}

	d.SetId(name)
	d.Set("id", name) // is this needed?

	return diags
}

func resourceAclPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	policy := d.Get("policy").(string)

	_, err := client.SystemACLPolicyUpdate(ctx, name, &rundeck.SystemACLPolicyUpdateRequest{
		Contents: &policy,
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call Client.SystemACLPolicyUpdate"))
	}

	return diags
}

func resourceAclPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()

	acl, err := client.SystemACLPolicyGet(ctx, name)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call client.SystemACLPolicyGet"))
	}
	if acl.StatusCode == 404 {
		return diag.FromErr(fmt.Errorf("ACL not found: (%s)", name))
	}

	d.Set("policy", acl.Contents)

	return diags
}

func resourceAclPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()

	_, err := client.SystemACLPolicyDelete(ctx, name)
	if err != nil {
		diag.FromErr(errors.Wrap(err, "failed to call client.SystemACLPolicyDelete"))
	}

	d.SetId("")

	return diags
}
