package rundeck

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckPrivateKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePrivateKeyCreateOrUpdate,
		UpdateContext: resourcePrivateKeyCreateOrUpdate,
		DeleteContext: resourcePrivateKeyDelete,
		ReadContext:   resourcePrivateKeyRead,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"key_material": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The private key material to store, in PEM format",
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},
		},
	}
}

func resourcePrivateKeyCreateOrUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	var diags diag.Diagnostics
	// var err error

	keyMaterialReader := ioutil.NopCloser(strings.NewReader(keyMaterial))

	if d.Id() != "" {
		resp, err := client.StorageKeyUpdate(ctx, path, keyMaterialReader, "application/octect-stream")
		if resp.StatusCode == 409 || err != nil {
			return diag.FromErr(errors.Wrap(err, "Error updating or adding key: Key exists"))
		}
	} else {
		resp, err := client.StorageKeyCreate(ctx, path, keyMaterialReader, "application/octet-stream")
		if resp.StatusCode == 409 || err != nil {
			return diag.FromErr(errors.Wrap(err, "Error updating or adding key: Key exists"))
		}
	}

	// if err != nil {
	// 	return err
	// }

	d.SetId(path)

	resourcePrivateKeyRead(ctx, d, meta)

	return diags
}

func resourcePrivateKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	var diags diag.Diagnostics
	path := d.Id()

	// The only "delete" call we have is oblivious to key type, but
	// that's okay since our Exists implementation makes sure that we
	// won't try to delete a key of the wrong type since we'll pretend
	// that it's already been deleted.

	resp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "client error getting StorageKeyGetMetadata from rundeck"))
	}

	if resp.StatusCode == 404 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Warning Message Summary",
			Detail:   fmt.Sprintf("no key found to delete on path (%s)", path),
		})
		d.SetId("")
		return diags
	}

	if resp.Meta.RundeckKeyType != rundeck.Private {
		// If the key type isn't private then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		d.SetId("")
		return diag.FromErr(fmt.Errorf("no private key found to delete on path (%s)", path))
	}

	_, err = client.StorageKeyDelete(ctx, path)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, fmt.Sprintf("Failed deleting the private key (%s)", path)))
	}

	d.SetId("")

	return diags
}

func resourcePrivateKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
