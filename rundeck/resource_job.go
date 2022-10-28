package rundeck

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckJob() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRundeckJobCreate,
		UpdateContext: resourceRundeckJobUpdate,
		DeleteContext: resourceRundeckJobDelete,
		ReadContext:   resourceRundeckJobRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Required: true,
			},

			"execution_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"log_level": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "INFO",
			},

			"allow_concurrent_executions": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"retry": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"max_thread_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"continue_on_error": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"continue_next_node_on_error": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"rank_order": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ascending",
			},

			"rank_attribute": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"success_on_empty_node_filter": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"preserve_options_order": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"command_ordering_strategy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "node-first",
			},

			"node_filter_query": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_filter_exclude_query": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_filter_exclude_precedence": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"timeout": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"schedule": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"schedule_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"time_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"notification": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Option of `on_success`, `on_failure`, `on_start`",
						},
						"email": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"attach_log": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"recipients": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"subject": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"webhook_urls": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"webhook_http_method": {
							Type:        schema.TypeString,
							Description: "One of `get`, `post`",
							Optional:    true,
						},
						"webhook_format": {
							Type:        schema.TypeString,
							Description: "One of `xml`, `json`",
							Optional:    true,
						},
						"plugin": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
						},
					},
				},
			},

			"option": {
				// This is a list because order is important when preserve_options_order is
				// set. When it's not set the order is unimportant but preserved by Rundeck/
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"label": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"default_value": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"value_choices": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"value_choices_url": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"require_predefined_choice": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"validation_regex": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"required": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"allow_multiple_values": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"multi_value_delimiter": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"obscure_input": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"exposed_to_scripts": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"storage_path": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"is_date": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"date_format": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Example: 'MM/DD/YYYY hh:mm a'. Should be as per momentjs",
						},
					},
				},
			},

			"global_log_filter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobFilter(),
			},

			"command": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     resourceRundeckJobCommand(),
			},
		},
	}
}

// Attention - Changes made to this function should be repeated in resourceRundeckJobCommandErrorHandler below!
func resourceRundeckJobCommand() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandJob(),
			},

			"step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},
			"node_step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},
			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"error_handler": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandErrorHandler(),
			},
		},
	}
}

// Terraform schemas do not support recursion. The Error Handler is a command within a command, but we're breaking it
// out and repeating it verbatim except for an inner errorHandler field.
func resourceRundeckJobCommandErrorHandler() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandJob(),
			},

			"step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},

			"node_step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},

			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobCommandScriptInterpreter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"invocation_string": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"args_quoted": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobCommandJob() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"group_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"run_for_each_node": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"args": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"node_filters": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"exclude_precedence": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"filter": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"exclude_filter": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceRundeckJobPluginResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobFilter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, importJobErr := importJob(client, job, "create")
	if importJobErr != nil {
		return diag.FromErr(errors.Wrap(importJobErr, "failed to import job"))
	}

	d.SetId(jobSummary.ID)

	return resourceRundeckJobRead(ctx, d, meta)
}

func resourceRundeckJobUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, importJobErr := importJob(client, job, "update")
	if err != nil {
		return diag.FromErr(errors.Wrap(importJobErr, "failed to import job"))
	}

	d.SetId(jobSummary.ID)

	return resourceRundeckJobRead(ctx, d, meta)
}

func resourceRundeckJobDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	_, err := client.JobDelete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to delete job"))
	}

	d.SetId("")

	return diags
}

func resourceRundeckJobRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*rundeck.BaseClient)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	job, err := GetJob(ctx, client, d.Id())
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to get job"))
	}

	diags = append(diags, jobToResourceData(job, d)...)

	return diags
}
