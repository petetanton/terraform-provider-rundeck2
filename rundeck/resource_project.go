package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,
		ReadContext:   resourceProjectRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name for the project",
				ForceNew:    true,
			},

			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the project to be shown in the Rundeck UI",
				Default:     "Managed by Terraform",
			},

			"ui_url": {
				Type:     schema.TypeString,
				Required: false,
				Computed: true,
			},

			"resource_model_source": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the resource model plugin to use",
						},
						"config": {
							Type:        schema.TypeMap,
							Required:    true,
							Description: "Configuration parameters for the selected plugin",
						},
					},
				},
			},

			"default_node_file_copier_plugin": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "jsch-scp",
			},

			"default_node_executor_plugin": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "jsch-ssh",
			},

			"ssh_authentication_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "privateKey",
			},

			"ssh_key_storage_path": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"ssh_key_file_path": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"extra_config": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional raw configuration parameters to include in the project configuration, with dots replaced with slashes in the key names due to limitations in Terraform's config language.",
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Rundeck's model is a little inconsistent in that we can create
	// a project via a high-level structure but yet we must update
	// the project via its raw config properties.
	// For simplicity's sake we create a bare minimum project here
	// and then delegate to UpdateProject to fill in the rest of the
	// configuration via the raw config properties.

	name := d.Get("name").(string)

	_, err := client.ProjectCreate(ctx, rundeck.ProjectCreateRequest{
		Name: &name,
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call Client.ProjectCreate"))
	}

	d.SetId(name)
	d.Set("name", name)

	return resourceProjectUpdate(ctx, d, meta)
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	// In Rundeck, updates are always in terms of the low-level config
	// properties map, so we need to transform our data structure
	// into the equivalent raw properties.

	projectName := d.Id()

	updateMap := map[string]string{}

	slashReplacer := strings.NewReplacer("/", ".")
	if extraConfig := d.Get("extra_config"); extraConfig != nil {
		for k, v := range extraConfig.(map[string]interface{}) {
			updateMap[slashReplacer.Replace(k)] = v.(string)
		}
	}

	for configKey, attrKey := range projectConfigAttributes {
		v := d.Get(attrKey).(string)
		if v != "" {
			updateMap[configKey] = v
		}
	}

	for i, rmsi := range d.Get("resource_model_source").([]interface{}) {
		rms := rmsi.(map[string]interface{})
		pluginType := rms["type"].(string)
		ci := rms["config"].(map[string]interface{})
		attrKeyPrefix := fmt.Sprintf("resources.source.%v.", i+1)
		typeKey := attrKeyPrefix + "type"
		configKeyPrefix := fmt.Sprintf("%vconfig.", attrKeyPrefix)
		updateMap[typeKey] = pluginType
		for k, v := range ci {
			updateMap[configKeyPrefix+k] = v.(string)
		}
	}

	_, err := client.ProjectConfigUpdate(ctx, projectName, updateMap)

	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call client.ProjectConfigUpdate"))
	}

	resourceProjectRead(ctx, d, meta)

	return diags
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	project, err := client.ProjectGet(ctx, name)

	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to call client.ProjectGet"))
	}

	if project.StatusCode == 404 {
		return diag.FromErr(fmt.Errorf("project not found: (%s)", name))
	}

	projectConfig := project.Config.(map[string]interface{})

	for configKey, attrKey := range projectConfigAttributes {
		d.Set(projectConfigAttributes[configKey], nil)
		if v, ok := projectConfig[configKey]; ok {
			d.Set(attrKey, v)
			// Remove this key so it won't get included in extra_config
			// later.
			delete(projectConfig, configKey)
		}
	}

	resourceSourceMap := map[int]interface{}{}
	configMaps := map[int]interface{}{}
	for configKey, v := range projectConfig {
		if strings.HasPrefix(configKey, "resources.source.") {
			nameParts := strings.Split(configKey, ".")

			if len(nameParts) < 4 {
				continue
			}

			index, err := strconv.Atoi(nameParts[2])
			if err != nil {
				continue
			}

			if _, ok := resourceSourceMap[index]; !ok {
				configMap := map[string]interface{}{}
				configMaps[index] = configMap
				resourceSourceMap[index] = map[string]interface{}{
					"config": configMap,
				}
			}

			switch nameParts[3] {
			case "type":
				if len(nameParts) != 4 {
					continue
				}
				m := resourceSourceMap[index].(map[string]interface{})
				m["type"] = v
			case "config":
				if len(nameParts) != 5 {
					continue
				}
				m := configMaps[index].(map[string]interface{})
				m[nameParts[4]] = v
			default:
				continue
			}

			// Remove this key so it won't get included in extra_config
			// later.
			delete(projectConfig, configKey)
		}
	}

	resourceSources := []map[string]interface{}{}
	resourceSourceIndices := []int{}
	for k := range resourceSourceMap {
		resourceSourceIndices = append(resourceSourceIndices, k)
	}
	sort.Ints(resourceSourceIndices)

	for _, index := range resourceSourceIndices {
		resourceSources = append(resourceSources, resourceSourceMap[index].(map[string]interface{}))
	}
	d.Set("resource_model_source", resourceSources)

	extraConfig := map[string]string{}
	dotReplacer := strings.NewReplacer(".", "/")
	for k, v := range projectConfig {
		extraConfig[dotReplacer.Replace(k)] = v.(string)
	}
	d.Set("extra_config", extraConfig)

	d.Set("name", project.Name)
	d.Set("ui_url", project.URL)

	return diags
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	_, err := client.ProjectDelete(ctx, name)
	if err != nil {
		diag.FromErr(errors.Wrap(err, "failed to call client.ProjectDelete"))
	}

	return diags
}
