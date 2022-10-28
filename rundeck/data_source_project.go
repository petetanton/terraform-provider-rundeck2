package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/rundeck/go-rundeck/rundeck"
)

var projectConfigAttributes = map[string]string{
	"project.name":                          "name",
	"project.description":                   "description",
	"service.FileCopier.default.provider":   "default_node_file_copier_plugin",
	"service.NodeExecutor.default.provider": "default_node_executor_plugin",
	"project.ssh-authentication":            "ssh_authentication_type",
	"project.ssh-key-storage-path":          "ssh_key_storage_path",
	"project.ssh-keypath":                   "ssh_key_file_path",
}

func dataSourcesProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name for the project",
				ForceNew:    true,
			},

			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the project to be shown in the Rundeck UI",
			},

			"ui_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"resource_model_source": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the resource model plugin to use",
						},
						"config": {
							Type:        schema.TypeMap,
							Computed:    true,
							Description: "Configuration parameters for the selected plugin",
						},
					},
				},
			},

			"default_node_file_copier_plugin": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_node_executor_plugin": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_authentication_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_key_storage_path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_key_file_path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"extra_config": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Additional raw configuration parameters to include in the project configuration, with dots replaced with slashes in the key names due to limitations in Terraform's config language.",
			},
		},
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	project, err := client.ProjectGet(ctx, name)
	if err != nil {
		return diag.FromErr(err)
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

	d.SetId(name)

	return diags
}
