package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/rundeck/go-rundeck/rundeck"
)

func dataSourceRundeckPrivateKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePrivateKeyRead,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},
		},
	}
}

func dataSourcePrivateKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	path := d.Id()

	resp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed getting StorageKeyGetMetadata from rundeck"))
	}

	if resp.StatusCode == 404 {
		return diag.FromErr(fmt.Errorf("received 404 error from StorageKeyGetMetadata for path (%s)", path))
	}

	if resp.Meta.RundeckKeyType != rundeck.Private {
		// If the key type isn't private then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return diag.FromErr(fmt.Errorf("failed finding a private key on path (%s)", path))
	}

	return diags
}
