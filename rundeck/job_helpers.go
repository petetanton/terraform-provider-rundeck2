package rundeck

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func jobFromResourceData(d *schema.ResourceData) (*JobDetail, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	job := &JobDetail{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		GroupName:                 d.Get("group_name").(string),
		ProjectName:               d.Get("project_name").(string),
		Description:               d.Get("description").(string),
		ExecutionEnabled:          d.Get("execution_enabled").(bool),
		Timeout:                   d.Get("timeout").(string),
		ScheduleEnabled:           d.Get("schedule_enabled").(bool),
		TimeZone:                  d.Get("time_zone").(string),
		LogLevel:                  d.Get("log_level").(string),
		AllowConcurrentExecutions: d.Get("allow_concurrent_executions").(bool),
		Retry:                     d.Get("retry").(string),
		Dispatch: &JobDispatch{
			MaxThreadCount:          d.Get("max_thread_count").(int),
			ContinueNextNodeOnError: d.Get("continue_next_node_on_error").(bool),
			RankAttribute:           d.Get("rank_attribute").(string),
			RankOrder:               d.Get("rank_order").(string),
		},
	}

	successOnEmpty := d.Get("success_on_empty_node_filter")
	if successOnEmpty != nil {
		job.Dispatch.SuccessOnEmptyNodeFilter = successOnEmpty.(bool)
	}

	sequence := &JobCommandSequence{
		ContinueOnError:  d.Get("continue_on_error").(bool),
		OrderingStrategy: d.Get("command_ordering_strategy").(string),
		Commands:         []JobCommand{},
	}

	logFilterConfigs := d.Get("global_log_filter").([]interface{})
	if len(logFilterConfigs) > 0 {
		globalLogFilters := &[]JobLogFilter{}
		for _, logFilterI := range logFilterConfigs {
			logFilterMap := logFilterI.(map[string]interface{})
			configI := logFilterMap["config"].(map[string]interface{})
			config := &JobLogFilterConfig{}
			for key, value := range configI {
				(*config)[key] = value.(string)
			}
			logFilter := &JobLogFilter{
				Type:   logFilterMap["type"].(string),
				Config: config,
			}

			*globalLogFilters =
				append(*globalLogFilters, *logFilter)
		}
		sequence.GlobalLogFilters = globalLogFilters
	}
	commandConfigs := d.Get("command").([]interface{})
	for _, commandI := range commandConfigs {
		command, err := commandFromResourceData(commandI)
		if err != nil {
			return nil, err
		}
		sequence.Commands = append(sequence.Commands, *command)
	}
	job.CommandSequence = sequence

	optionConfigsI := d.Get("option").([]interface{})
	if len(optionConfigsI) > 0 {
		optionsConfig := &JobOptions{
			PreserveOrder: d.Get("preserve_options_order").(bool),
			Options:       []JobOption{},
		}
		for _, optionI := range optionConfigsI {
			optionMap := optionI.(map[string]interface{})
			option := JobOption{
				Name:                    optionMap["name"].(string),
				Label:                   optionMap["label"].(string),
				DefaultValue:            optionMap["default_value"].(string),
				ValueChoices:            JobValueChoices([]string{}),
				ValueChoicesURL:         optionMap["value_choices_url"].(string),
				RequirePredefinedChoice: optionMap["require_predefined_choice"].(bool),
				ValidationRegex:         optionMap["validation_regex"].(string),
				Description:             optionMap["description"].(string),
				IsRequired:              optionMap["required"].(bool),
				AllowsMultipleValues:    optionMap["allow_multiple_values"].(bool),
				MultiValueDelimiter:     optionMap["multi_value_delimiter"].(string),
				ObscureInput:            optionMap["obscure_input"].(bool),
				ValueIsExposedToScripts: optionMap["exposed_to_scripts"].(bool),
				StoragePath:             optionMap["storage_path"].(string),
				IsDate:                  optionMap["is_date"].(bool),
				DateFormat:              optionMap["date_format"].(string),
			}
			if option.StoragePath != "" && option.ObscureInput == false {
				return nil, diag.FromErr(fmt.Errorf("argument \"obscure_input\" must be set to `true` when \"storage_path\" is not empty"))
			}
			if option.ValueIsExposedToScripts && option.ObscureInput == false {
				return nil, diag.FromErr(fmt.Errorf("argument \"obscure_input\" must be set to `true` when \"exposed_to_scripts\" is set to true"))
			}
			if option.IsDate && option.DateFormat == "" {
				return nil, diag.FromErr(fmt.Errorf("if \"is_data\" is set, you must set \"date_format\" (in momentjs)"))
			}

			for _, iv := range optionMap["value_choices"].([]interface{}) {
				if iv == nil {
					return nil, diag.FromErr(fmt.Errorf("argument \"value_choices\" can not have empty values; try \"required\""))
				}
				option.ValueChoices = append(option.ValueChoices, iv.(string))
			}

			optionsConfig.Options = append(optionsConfig.Options, option)
		}
		job.OptionsConfig = optionsConfig
	}

	job.NodeFilter = &JobNodeFilter{
		ExcludePrecedence: d.Get("node_filter_exclude_precedence").(bool),
	}
	if nodeFilterQuery := d.Get("node_filter_query").(string); nodeFilterQuery != "" {
		job.NodeFilter.Query = nodeFilterQuery
	}
	if nodeFilterExcludeQuery := d.Get("node_filter_exclude_query").(string); nodeFilterExcludeQuery != "" {
		job.NodeFilter.ExcludeQuery = nodeFilterExcludeQuery
	}

	if err := JobScheduleFromResourceData(d, job); err != nil {
		return nil, err
	}

	notificationsConfigI := d.Get("notification").([]interface{})
	if len(notificationsConfigI) > 0 {
		if len(notificationsConfigI) <= 3 {
			jobNotification := JobNotification{}
			// test if unique
			for _, notificationI := range notificationsConfigI {
				notification := Notification{}
				notificationMap := notificationI.(map[string]interface{})
				jobType := notificationMap["type"].(string)

				// Get email notification data
				notificationEmailsI := notificationMap["email"].([]interface{})
				if len(notificationEmailsI) > 0 {
					notificationEmailI := notificationEmailsI[0].(map[string]interface{})
					email := EmailNotification{
						AttachLog:  notificationEmailI["attach_log"].(bool),
						Recipients: NotificationEmails([]string{}),
						Subject:    notificationEmailI["subject"].(string),
					}
					for _, iv := range notificationEmailI["recipients"].([]interface{}) {
						email.Recipients = append(email.Recipients, iv.(string))
					}
					notification.Email = &email
				}

				// Webhook notification
				webHookUrls := notificationMap["webhook_urls"].([]interface{})
				if len(webHookUrls) > 0 {
					webHook := &WebHookNotification{
						Urls: NotificationUrls([]string{}),
					}
					for _, iv := range webHookUrls {
						webHook.Urls = append(webHook.Urls, iv.(string))
					}
					notification.WebHook = webHook
					notification.Format = notificationMap["webhook_format"].(string)
					notification.HttpMethod = notificationMap["webhook_http_method"].(string)
				}

				// plugin Notification
				notificationPluginsI := notificationMap["plugin"].([]interface{})
				if len(notificationPluginsI) > 1 {
					return nil, diag.FromErr(fmt.Errorf("rundeck command may have no more than one notification plugin"))
				}
				if len(notificationPluginsI) > 0 {
					notificationPluginMap := notificationPluginsI[0].(map[string]interface{})
					configI := notificationPluginMap["config"].(map[string]interface{})
					config := map[string]string{}
					for k, v := range configI {
						config[k] = v.(string)
					}
					notification.Plugin = &JobPlugin{
						Type:   notificationPluginMap["type"].(string),
						Config: config,
					}
				}

				switch jobType {
				case "on_success":
					if jobNotification.OnSuccess != nil {
						return nil, diag.FromErr(fmt.Errorf("a block with %s already exists", jobType))
					}
					jobNotification.OnSuccess = &notification
					job.Notification = &jobNotification
				case "on_failure":
					if jobNotification.OnFailure != nil {
						return nil, diag.FromErr(fmt.Errorf("a block with %s already exists", jobType))
					}
					jobNotification.OnFailure = &notification
					job.Notification = &jobNotification
				case "on_start":
					if jobNotification.OnStart != nil {
						return nil, diag.FromErr(fmt.Errorf("a block with %s already exists", jobType))
					}
					jobNotification.OnStart = &notification
					job.Notification = &jobNotification
				default:
					return nil, diag.FromErr(fmt.Errorf("the notification type is not one of `on_success`, `on_failure`, `on_start`"))
				}
			}
		} else {
			return nil, diag.FromErr(fmt.Errorf("can only have up to three notification blocks, `on_success`, `on_failure`, `on_start`"))
		}
	}
	return job, diags
}

func jobToResourceData(job *JobDetail, d *schema.ResourceData) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	d.SetId(job.ID)
	if err := d.Set("name", job.Name); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `name`"))
	}
	if err := d.Set("group_name", job.GroupName); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `group_name`"))
	}

	// The project name is not consistently returned in all rundeck versions,
	// so we'll only update it if it's set. Jobs can't move between projects
	// anyway, so this is harmless.
	if job.ProjectName != "" {
		if err := d.Set("project_name", job.ProjectName); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `project_name`"))
		}
	}

	if err := d.Set("description", job.Description); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `description`"))
	}
	if err := d.Set("execution_enabled", job.ExecutionEnabled); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `execution_enabled`"))
	}
	if err := d.Set("schedule_enabled", job.ScheduleEnabled); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `schedule_enabled`"))
	}
	if err := d.Set("time_zone", job.TimeZone); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `time_zone`"))
	}
	if err := d.Set("log_level", job.LogLevel); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `log_level`"))
	}
	if err := d.Set("allow_concurrent_executions", job.AllowConcurrentExecutions); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `allow_concurrent_executions`"))
	}
	if err := d.Set("retry", job.Retry); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `retry`"))
	}

	if job.Dispatch != nil {
		if err := d.Set("max_thread_count", job.Dispatch.MaxThreadCount); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `max_thread_count`"))
		}
		if err := d.Set("continue_next_node_on_error", job.Dispatch.ContinueNextNodeOnError); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `continue_next_node_on_error`"))
		}
		if err := d.Set("rank_attribute", job.Dispatch.RankAttribute); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `rank_attribute`"))
		}
		if err := d.Set("rank_order", job.Dispatch.RankOrder); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `rank_order`"))
		}
	} else {
		if err := d.Set("max_thread_count", 1); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `max_thread_count`"))
		}
		if err := d.Set("continue_next_node_on_error", false); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `continue_next_node_on_error`"))
		}
		if err := d.Set("rank_attribute", nil); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `rank_attribute`"))
		}
		if err := d.Set("rank_order", "ascending"); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `rank_order`"))
		}
	}

	if job.NodeFilter != nil {
		if err := d.Set("node_filter_query", job.NodeFilter.Query); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_query`"))
		}
		if err := d.Set("node_filter_exclude_query", job.NodeFilter.ExcludeQuery); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_exclude_query`"))
		}
		if err := d.Set("node_filter_exclude_precedence", job.NodeFilter.ExcludePrecedence); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_exclude_precedence`"))
		}
	} else {
		if err := d.Set("node_filter_query", nil); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_query`"))
		}
		if err := d.Set("node_filter_exclude_query", nil); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_exclude_query`"))
		}
		if err := d.Set("node_filter_exclude_precedence", nil); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `node_filter_exclude_precedence`"))
		}
	}

	optionConfigsI := make([]interface{}, 0)
	if job.OptionsConfig != nil {
		if err := d.Set("preserve_options_order", job.OptionsConfig.PreserveOrder); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `preserve_options_order`"))
		}
		for _, option := range job.OptionsConfig.Options {
			optionConfigI := map[string]interface{}{
				"name":                      option.Name,
				"label":                     option.Label,
				"default_value":             option.DefaultValue,
				"value_choices":             option.ValueChoices,
				"value_choices_url":         option.ValueChoicesURL,
				"require_predefined_choice": option.RequirePredefinedChoice,
				"validation_regex":          option.ValidationRegex,
				"description":               option.Description,
				"required":                  option.IsRequired,
				"allow_multiple_values":     option.AllowsMultipleValues,
				"multi_value_delimiter":     option.MultiValueDelimiter,
				"obscure_input":             option.ObscureInput,
				"exposed_to_scripts":        option.ValueIsExposedToScripts,
				"storage_path":              option.StoragePath,
			}
			optionConfigsI = append(optionConfigsI, optionConfigI)
		}
	}
	if err := d.Set("option", optionConfigsI); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `option`"))
	}

	if job.CommandSequence != nil {
		if err := d.Set("command_ordering_strategy", job.CommandSequence.OrderingStrategy); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `command_ordering_strategy`"))
		}
		if err := d.Set("continue_on_error", job.CommandSequence.ContinueOnError); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `continue_on_error`"))
		}

		if job.CommandSequence.GlobalLogFilters != nil && len(*job.CommandSequence.GlobalLogFilters) > 0 {
			globalLogFilterConfigsI := make([]interface{}, 0)
			for _, logFilter := range *job.CommandSequence.GlobalLogFilters {
				logFilterI := map[string]interface{}{
					"type":   logFilter.Type,
					"config": map[string]string(*logFilter.Config),
				}
				globalLogFilterConfigsI = append(globalLogFilterConfigsI, logFilterI)
			}
			if err := d.Set("global_log_filter", globalLogFilterConfigsI); err != nil {
				return diag.FromErr(errors.Wrap(err, "failed to set `global_log_filter`"))
			}
		}

		commandConfigsI := make([]interface{}, 0)
		for i := range job.CommandSequence.Commands {
			commandConfigI, err := commandToResourceData(&job.CommandSequence.Commands[i])
			if err != nil {
				return err
			}
			commandConfigsI = append(commandConfigsI, commandConfigI)
		}
		if err := d.Set("command", commandConfigsI); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `command`"))
		}
	}

	if job.Schedule != nil {
		cronSpec, err := scheduleToCronSpec(job.Schedule)
		if err != nil {
			return err
		}
		if err := d.Set("schedule", cronSpec); err != nil {
			return diag.FromErr(errors.Wrap(err, "failed to set `schedule`"))
		}
	}
	notificationConfigsI := make([]interface{}, 0)
	if job.Notification != nil {
		if job.Notification.OnSuccess != nil {
			notificationConfigI := readNotification(job.Notification.OnSuccess, "on_success")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
		if job.Notification.OnFailure != nil {
			notificationConfigI := readNotification(job.Notification.OnFailure, "on_failure")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
		if job.Notification.OnStart != nil {
			notificationConfigI := readNotification(job.Notification.OnStart, "on_start")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
	}

	if err := d.Set("notification", notificationConfigsI); err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to set `notification`"))
	}

	return diags
}

func JobScheduleFromResourceData(d *schema.ResourceData, job *JobDetail) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	const scheduleKey = "schedule"
	cronSpec := d.Get(scheduleKey).(string)
	if cronSpec != "" {
		schedule := strings.Split(cronSpec, " ")
		if len(schedule) != 7 {
			return diag.FromErr(fmt.Errorf("the Rundeck schedule must be formatted like a cron expression, as defined here: http://www.quartz-scheduler.org/documentation/quartz-2.2.x/tutorials/tutorial-lesson-06.html"))
		}
		job.Schedule = &JobSchedule{
			Time: JobScheduleTime{
				Seconds: schedule[0],
				Minute:  schedule[1],
				Hour:    schedule[2],
			},
			Month: JobScheduleMonth{
				Day:   schedule[3],
				Month: schedule[4],
			},
			WeekDay: JobScheduleWeekDay{
				Day: schedule[5],
			},
			Year: JobScheduleYear{
				Year: schedule[6],
			},
		}
		// Day-of-month and Day-of-week can both be asterisks, but otherwise one, and only one, must be a '?'
		if job.Schedule.Month.Day == job.Schedule.WeekDay.Day {
			if job.Schedule.Month.Day != "*" {
				return diag.FromErr(fmt.Errorf("invalid '%s' specification %s - one of day-of-month (4th item) or day-of-week (6th) must be '?'", scheduleKey, cronSpec))
			}
		} else if job.Schedule.Month.Day != "?" && job.Schedule.WeekDay.Day != "?" {
			return diag.FromErr(fmt.Errorf("invalid '%s' specification %s - one of day-of-month (4th item) or day-of-week (6th) must be '?'", scheduleKey, cronSpec))
		}
	}
	return diags
}

func scheduleToCronSpec(schedule *JobSchedule) (string, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if schedule.Month.Day == "" {
		if schedule.WeekDay.Day == "*" || schedule.WeekDay.Day == "" {
			schedule.Month.Day = "*"
		} else {
			schedule.Month.Day = "?"
		}
	}
	if schedule.WeekDay.Day == "" {
		if schedule.Month.Day == "*" {
			schedule.WeekDay.Day = "*"
		} else {
			schedule.WeekDay.Day = "?"
		}
	}
	cronSpec := make([]string, 0)
	cronSpec = append(cronSpec, schedule.Time.Seconds)
	cronSpec = append(cronSpec, schedule.Time.Minute)
	cronSpec = append(cronSpec, schedule.Time.Hour)
	cronSpec = append(cronSpec, schedule.Month.Day)
	cronSpec = append(cronSpec, schedule.Month.Month)
	cronSpec = append(cronSpec, schedule.WeekDay.Day)
	cronSpec = append(cronSpec, schedule.Year.Year)
	return strings.Join(cronSpec, " "), diags
}

func commandFromResourceData(commandI interface{}) (*JobCommand, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	commandMap := commandI.(map[string]interface{})
	command := &JobCommand{
		Description:        commandMap["description"].(string),
		ShellCommand:       commandMap["shell_command"].(string),
		Script:             commandMap["inline_script"].(string),
		ScriptFile:         commandMap["script_file"].(string),
		ScriptFileArgs:     commandMap["script_file_args"].(string),
		KeepGoingOnSuccess: commandMap["keep_going_on_success"].(bool),
	}

	// Because of the lack of schema recursion, the inner command has a separate schema without an error_handler
	// field, but is otherwise identical. The 'exists' checks allow this function to apply to both 'command' and
	// 'errorHandler' schemas.
	if errorHandlersI, exists := commandMap["error_handler"].([]interface{}); exists {
		if len(errorHandlersI) > 1 {
			return nil, diag.FromErr(fmt.Errorf("rundeck command may have no more than one error handler"))
		}
		if len(errorHandlersI) > 0 {
			errorHandlerMap := errorHandlersI[0].(map[string]interface{})
			errorHandler, err := commandFromResourceData(errorHandlerMap)
			if err != nil {
				return nil, err
			}
			command.ErrorHandler = errorHandler
		}
	}

	scriptInterpretersI := commandMap["script_interpreter"].([]interface{})
	if len(scriptInterpretersI) > 1 {
		return nil, diag.FromErr(fmt.Errorf("rundeck command may have no more than one script interpreter"))
	}
	if len(scriptInterpretersI) > 0 {
		scriptInterpreterMap := scriptInterpretersI[0].(map[string]interface{})
		command.ScriptInterpreter = &JobCommandScriptInterpreter{
			InvocationString: scriptInterpreterMap["invocation_string"].(string),
			ArgsQuoted:       scriptInterpreterMap["args_quoted"].(bool),
		}
	}

	var err diag.Diagnostics
	if command.Job, err = jobCommandJobRefFromResourceData("job", commandMap); err != nil {
		return nil, err
	}
	if command.StepPlugin, err = singlePluginFromResourceData("step_plugin", commandMap); err != nil {
		return nil, err
	}
	if command.NodeStepPlugin, err = singlePluginFromResourceData("node_step_plugin", commandMap); err != nil {
		return nil, err
	}

	return command, diags
}

func jobCommandJobRefFromResourceData(key string, commandMap map[string]interface{}) (*JobCommandJobRef, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	jobRefsI := commandMap[key].([]interface{})
	if len(jobRefsI) > 1 {
		return nil, diag.FromErr(fmt.Errorf("rundeck command may have no more than one %s", key))
	}
	if len(jobRefsI) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Command Job with key: \"%s\" returned no results", key),
			Detail:   "Returning nil",
		})
		return nil, diags
	}
	jobRefMap := jobRefsI[0].(map[string]interface{})
	jobRef := &JobCommandJobRef{
		Name:           jobRefMap["name"].(string),
		GroupName:      jobRefMap["group_name"].(string),
		RunForEachNode: jobRefMap["run_for_each_node"].(bool),
		Arguments:      JobCommandJobRefArguments(jobRefMap["args"].(string)),
	}
	nodeFiltersI := jobRefMap["node_filters"].([]interface{})
	if len(nodeFiltersI) > 1 {
		return nil, diag.FromErr(fmt.Errorf("rundeck command job reference may have no more than one node filter"))
	}
	if len(nodeFiltersI) > 0 {
		nodeFilterMap := nodeFiltersI[0].(map[string]interface{})
		jobRef.NodeFilter = &JobNodeFilter{
			Query:             nodeFilterMap["filter"].(string),
			ExcludeQuery:      nodeFilterMap["exclude_filter"].(string),
			ExcludePrecedence: nodeFilterMap["exclude_precedence"].(bool),
		}
	}
	return jobRef, diags
}

func singlePluginFromResourceData(key string, commandMap map[string]interface{}) (*JobPlugin, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	stepPluginsI := commandMap[key].([]interface{})
	if len(stepPluginsI) > 1 {
		return nil, diag.FromErr(fmt.Errorf("rundeck command may have no more than one %s", key))
	}
	if len(stepPluginsI) == 0 {
		return nil, nil
	}
	stepPluginMap := stepPluginsI[0].(map[string]interface{})
	configI := stepPluginMap["config"].(map[string]interface{})
	config := map[string]string{}
	for key, value := range configI {
		config[key] = value.(string)
	}
	result := &JobPlugin{
		Type:   stepPluginMap["type"].(string),
		Config: config,
	}
	return result, diags
}

func commandToResourceData(command *JobCommand) (map[string]interface{}, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	commandConfigI := map[string]interface{}{
		"description":           command.Description,
		"shell_command":         command.ShellCommand,
		"inline_script":         command.Script,
		"script_file":           command.ScriptFile,
		"script_file_args":      command.ScriptFileArgs,
		"keep_going_on_success": command.KeepGoingOnSuccess,
	}

	if command.ErrorHandler != nil {
		errorHandlerI, err := commandToResourceData(command.ErrorHandler)
		if err != nil {
			return nil, err
		}
		commandConfigI["error_handler"] = []interface{}{
			errorHandlerI,
		}
	}

	if command.ScriptInterpreter != nil {
		commandConfigI["script_interpreter"] = []interface{}{
			map[string]interface{}{
				"invocation_string": command.ScriptInterpreter.InvocationString,
				"args_quoted":       command.ScriptInterpreter.ArgsQuoted,
			},
		}
	}

	if command.Job != nil {
		jobRefConfigI := map[string]interface{}{
			"name":              command.Job.Name,
			"group_name":        command.Job.GroupName,
			"run_for_each_node": command.Job.RunForEachNode,
			"args":              command.Job.Arguments,
		}
		if command.Job.NodeFilter != nil {
			nodeFilterConfigI := map[string]interface{}{
				"exclude_precedence": command.Job.NodeFilter.ExcludePrecedence,
				"filter":             command.Job.NodeFilter.Query,
				"exclude_filter":     command.Job.NodeFilter.ExcludeQuery,
			}
			jobRefConfigI["node_filters"] = append([]interface{}{}, nodeFilterConfigI)
		}
		commandConfigI["job"] = append([]interface{}{}, jobRefConfigI)
	}

	if command.StepPlugin != nil {
		commandConfigI["step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.StepPlugin.Type,
				"config": map[string]string(command.StepPlugin.Config),
			},
		}
	}

	if command.NodeStepPlugin != nil {
		commandConfigI["node_step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.NodeStepPlugin.Type,
				"config": map[string]string(command.NodeStepPlugin.Config),
			},
		}
	}
	return commandConfigI, diags
}

// Helper function for three different notifications
func readNotification(notification *Notification, notificationType string) map[string]interface{} {
	notificationConfigI := map[string]interface{}{
		"type": notificationType,
	}
	if notification.WebHook != nil {
		notificationConfigI["webhook_urls"] = notification.WebHook.Urls
		notificationConfigI["webhook_http_method"] = notification.HttpMethod
		notificationConfigI["webhook_format"] = notification.Format
	}
	if notification.Email != nil {
		notificationConfigI["email"] = []interface{}{
			map[string]interface{}{
				"attach_log": notification.Email.AttachLog,
				"subject":    notification.Email.Subject,
				"recipients": notification.Email.Recipients,
			},
		}
	}
	if notification.Plugin != nil {
		notificationConfigI["plugin"] = []interface{}{
			map[string]interface{}{
				"type":   notification.Plugin.Type,
				"config": map[string]string(notification.Plugin.Config),
			},
		}
	}
	return notificationConfigI
}
