package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/rundeck/go-rundeck/rundeck"
)

func dataSourceRundeckJob() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRundeckJobRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"execution_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"log_level": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"allow_concurrent_executions": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"retry": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"max_thread_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"continue_on_error": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"continue_next_node_on_error": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"rank_order": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"rank_attribute": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"success_on_empty_node_filter": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"preserve_options_order": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"command_ordering_strategy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"node_filter_query": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"node_filter_exclude_query": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"node_filter_exclude_precedence": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"timeout": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"schedule": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"schedule_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"time_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"notification": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Option of `on_success`, `on_failure`, `on_start`",
						},
						"email": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"attach_log": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"recipients": {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"subject": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"webhook_urls": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"webhook_http_method": {
							Type:        schema.TypeString,
							Description: "One of `get`, `post`",
							Computed:    true,
						},
						"webhook_format": {
							Type:        schema.TypeString,
							Description: "One of `xml`, `json`",
							Computed:    true,
						},
						"plugin": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     dataSourceRundeckJobPluginResource(),
						},
					},
				},
			},

			"option": {
				// This is a list because order is important when preserve_options_order is
				// set. When it's not set the order is unimportant but preserved by Rundeck/
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"default_value": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"value_choices": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"value_choices_url": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"require_predefined_choice": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"validation_regex": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"required": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"allow_multiple_values": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"multi_value_delimiter": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"obscure_input": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"exposed_to_scripts": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"storage_path": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"is_date": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"date_format": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Example: 'MM/DD/YYYY hh:mm a'. Should be as per momentjs",
						},
					},
				},
			},

			"global_log_filter": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobFilter(),
			},

			"command": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommand(),
			},
		},
	}
}

// Attention - Changes made to this function should be repeated in resourceRundeckJobCommandErrorHandler below!
func dataSourceRundeckJobCommand() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommandJob(),
			},

			"step_plugin": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobPluginResource(),
			},
			"node_step_plugin": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobPluginResource(),
			},
			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"error_handler": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommandErrorHandler(),
			},
		},
	}
}

// Terraform schemas do not support recursion. The Error Handler is a command within a command, but we're breaking it
// out and repeating it verbatim except for an inner errorHandler field.
func dataSourceRundeckJobCommandErrorHandler() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobCommandJob(),
			},

			"step_plugin": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobPluginResource(),
			},

			"node_step_plugin": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     dataSourceRundeckJobPluginResource(),
			},

			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceRundeckJobCommandScriptInterpreter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"invocation_string": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"args_quoted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceRundeckJobCommandJob() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"run_for_each_node": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"args": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"node_filters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"exclude_precedence": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"filter": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"exclude_filter": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRundeckJobPluginResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceRundeckJobFilter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceRundeckJobRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	id := d.Get("id").(string)

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("id: %s", id),
	})

	job, err := GetJob(ctx, client, id)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to get job from client"))
	}

	diags = append(diags, jobToResourceData(job, d)...)

	return diags
}
