package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"mediakit-cli/internal/cliexit"
	"mediakit-cli/internal/modes"
	"mediakit-cli/internal/notice"
	"mediakit-cli/internal/surface"
)

const taskIDDescription = "异步任务的唯一标识，用于查询任务状态并获取最终结果。"
const queryTaskInputGuidance = "task_id 必须来自真实异步任务受理结果；可选轮询参数仅在用户明确指定，或可从用户意图准确确定时填写，不得伪造。"
const queryTaskDescription = "查询异步任务状态与结果"
const sharedDomainDescription = "通用能力与任务查询"

func printDomains(cmd *cobra.Command) error {
	manifest := surface.Current()
	for _, domain := range manifest.Domains {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s\t%d\t%s\n",
			domain.Name,
			domain.CapabilityCount,
			domain.Description,
		); err != nil {
			return err
		}
	}
	if !modes.LocalSurfaceVisible() {
		return nil
	}
	for _, support := range manifest.SupportTools {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s\t%d\t%s\n",
			support.Domain,
			1,
			support.DomainDescription,
		); err != nil {
			return err
		}
	}
	return nil
}

func printHelpFull(cmd *cobra.Command) error {
	manifest := surface.Current()
	grouped := capabilitiesByDomain(manifest.Capabilities)
	lines := []string{
		"MediaKit CLI Full Help",
		"",
		"[shared]",
		sharedDomainDescription,
	}
	if modes.LocalSurfaceVisible() {
		supportTools := append([]surface.SupportTool(nil), manifest.SupportTools...)
		sort.Slice(supportTools, func(i int, j int) bool {
			return supportTools[i].Name < supportTools[j].Name
		})
		for _, support := range supportTools {
			capability := supportCapability(support)
			lines = append(
				lines,
				fmt.Sprintf("- %s    %s", capability.Name, capability.Description),
				fmt.Sprintf(
					"  查看详情: mediakit-cli %s %s --help",
					capability.Domain,
					capability.Name,
				),
			)
		}
	}
	lines = append(
		lines,
		fmt.Sprintf("- query-task    %s", queryTaskDescription),
		"  查看详情: mediakit-cli shared query-task --help",
		"",
	)
	for _, domain := range manifest.Domains {
		lines = append(
			lines,
			fmt.Sprintf("[%s]", domain.Name),
			domain.Description,
		)
		for _, capability := range grouped[domain.Name] {
			lines = append(
				lines,
				fmt.Sprintf("- %s    %s", capability.Name, capability.Description),
				fmt.Sprintf(
					"  查看详情: mediakit-cli %s %s --help",
					capability.Domain,
					capability.Name,
				),
			)
		}
		lines = append(lines, "")
	}
	_, err := fmt.Fprintln(
		cmd.OutOrStdout(),
		strings.TrimRight(strings.Join(lines, "\n"), "\n"),
	)
	return err
}

func newGeneratedDomainCommands() []*cobra.Command {
	manifest := surface.Current()
	grouped := capabilitiesByDomain(manifest.Capabilities)
	commands := make([]*cobra.Command, 0, len(manifest.Domains)+1)
	for _, domain := range manifest.Domains {
		currentDomain := domain
		domainCommand := &cobra.Command{
			Use:               currentDomain.Name,
			Short:             currentDomain.Description,
			Args:              cobra.NoArgs,
			DisableAutoGenTag: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
		}
		for _, capability := range grouped[currentDomain.Name] {
			domainCommand.AddCommand(newCapabilityCommand(capability))
		}
		configureDomainHelp(
			domainCommand,
			currentDomain,
			grouped[currentDomain.Name],
		)
		commands = append(commands, domainCommand)
	}
	commands = append(commands, newTaskProtocolCommand(manifest.SupportTools))
	return commands
}

func capabilitiesByDomain(capabilities []surface.Capability) map[string][]surface.Capability {
	grouped := map[string][]surface.Capability{}
	for _, capability := range capabilities {
		grouped[capability.Domain] = append(grouped[capability.Domain], capability)
	}
	for domain := range grouped {
		sort.Slice(grouped[domain], func(i int, j int) bool {
			return grouped[domain][i].Name < grouped[domain][j].Name
		})
	}
	return grouped
}

func renderDomainHelp(domain surface.Domain, capabilities []surface.Capability) string {
	lines := []string{
		fmt.Sprintf("%s — %s", domain.Name, domain.Description),
		"",
		"Available commands:",
	}
	for _, capability := range capabilities {
		lines = append(
			lines,
			fmt.Sprintf("- %s    %s", capability.Name, capability.Description),
			fmt.Sprintf("  查看详情: mediakit-cli %s %s --help", capability.Domain, capability.Name),
		)
	}
	return strings.Join(lines, "\n")
}

func configureDomainHelp(
	cmd *cobra.Command,
	domain surface.Domain,
	capabilities []surface.Capability,
) {
	cmd.SetHelpFunc(func(helpCommand *cobra.Command, args []string) {
		helpCommand.InitDefaultHelpFlag()
		lines := []string{
			strings.TrimRight(renderDomainHelp(domain, capabilities), "\n"),
			"",
			"Usage:",
			"  " + helpCommand.CommandPath() + " [flags]",
			"  " + helpCommand.CommandPath() + " [command]",
		}
		if usages := strings.TrimRight(
			helpCommand.LocalFlags().FlagUsages(),
			"\n",
		); usages != "" {
			lines = append(lines, "", "Flags:", usages)
		}
		if usages := strings.TrimRight(
			helpCommand.InheritedFlags().FlagUsages(),
			"\n",
		); usages != "" {
			lines = append(lines, "", "Global Flags:", usages)
		}
		lines = append(
			lines,
			"",
			fmt.Sprintf(
				"Use %q for more information about a command.",
				helpCommand.CommandPath()+" [command] --help",
			),
		)
		fmt.Fprintln(
			helpCommand.OutOrStdout(),
			strings.TrimRight(strings.Join(lines, "\n"), "\n"),
		)
	})
}

func newCapabilityCommand(capability surface.Capability) *cobra.Command {
	current := capability
	cmd := &cobra.Command{
		Use:               current.Name,
		Short:             current.Description,
		Args:              capabilityArgsValidator(current),
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
				schema, err := buildCapabilitySchema(cmd, current)
				if err != nil {
					return writeCapabilityError(cmd.OutOrStdout(), err)
				}
				return writeJSON(cmd.OutOrStdout(), schema)
			}
			params, err := collectCapabilityParams(cmd, current)
			if err != nil {
				return writeCapabilityError(cmd.OutOrStdout(), err)
			}
			runtimeMeta := capabilityRuntimeMeta(current)
			runtimeMeta.LocalUnsupportedParams = localUnsupportedParams(
				current,
				params,
			)
			if err := modes.Dispatch(cmd, runtimeMeta, params); err != nil {
				if errors.Is(err, cliexit.ErrBusinessFailure) {
					return err
				}
				return writeCapabilityError(cmd.OutOrStdout(), err)
			}
			return nil
		},
	}
	bindCapabilityFlags(cmd, current)
	if current.Local {
		cmd.Flags().String(
			"output-path",
			"",
			"本地文件输出目录或完整输出文件路径（覆盖 config/env 设置）",
		)
	}
	strictBool(cmd.Flags(), "schema", false, "输出该工具的 JSON Schema 描述（供 Agent 使用）")
	configureCapabilityHelp(cmd, current)
	return cmd
}

func newTaskProtocolCommand(supportTools []surface.SupportTool) *cobra.Command {
	localSurfaceVisible := modes.LocalSurfaceVisible()
	shared := &cobra.Command{
		Use:               "shared",
		Short:             sharedDomainDescription,
		DisableAutoGenTag: true,
	}
	for _, support := range supportTools {
		command := newCapabilityCommand(supportCapability(support))
		command.Hidden = !localSurfaceVisible
		shared.AddCommand(command)
	}
	query := &cobra.Command{
		Use:               "query-task",
		Short:             queryTaskDescription,
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if forceLocal {
				return writeCapabilityError(
					cmd.OutOrStdout(),
					fmt.Errorf("query-task 不支持 Local 模式，请使用 --cloud"),
				)
			}
			schemaOnly, _ := cmd.Flags().GetBool("schema")
			if schemaOnly {
				return writeJSON(cmd.OutOrStdout(), queryTaskSchema())
			}
			taskID, _ := cmd.Flags().GetString("task-id")
			if strings.TrimSpace(taskID) == "" {
				return writeCapabilityError(
					cmd.OutOrStdout(),
					fmt.Errorf("--task-id 为必填参数"),
				)
			}
			params := map[string]any{"task_id": taskID}
			queryInputSchema, _ := queryTaskSchema()["input_schema"].(map[string]any)
			queryProperties := schemaProperties(queryInputSchema)
			for _, item := range []struct {
				flag string
				name string
				read func() (any, error)
			}{
				{
					flag: "poll-interval-seconds",
					name: "poll_interval_seconds",
					read: func() (any, error) {
						return cmd.Flags().GetFloat64("poll-interval-seconds")
					},
				},
				{
					flag: "max-poll-attempts",
					name: "max_poll_attempts",
					read: func() (any, error) {
						return cmd.Flags().GetInt("max-poll-attempts")
					},
				},
				{
					flag: "poll-complete",
					name: "poll_complete",
					read: func() (any, error) {
						return cmd.Flags().GetBool("poll-complete")
					},
				},
				{
					flag: "max-poll-timeout-seconds",
					name: "max_poll_timeout_seconds",
					read: func() (any, error) {
						return cmd.Flags().GetFloat64("max-poll-timeout-seconds")
					},
				},
			} {
				if !cmd.Flags().Changed(item.flag) {
					continue
				}
				value, err := item.read()
				if err != nil {
					return err
				}
				if schema, ok := queryProperties[item.name]; ok {
					if err := validateSchemaValue(
						schema,
						value,
						item.name,
					); err != nil {
						return writeCapabilityError(
							cmd.OutOrStdout(),
							err,
						)
					}
				}
				params[item.name] = value
			}
			err := modes.Dispatch(
				cmd,
				modes.CapabilityRuntimeMeta{
					Name:        "query-task",
					Domain:      "shared",
					Description: queryTaskDescription,
					CloudOnly:   true,
				},
				params,
			)
			if err != nil && !errors.Is(err, cliexit.ErrBusinessFailure) {
				return writeCapabilityError(cmd.OutOrStdout(), err)
			}
			return err
		},
	}
	query.Flags().String("task-id", "", taskIDDescription)
	query.Flags().Float64("poll-interval-seconds", 10, "轮询间隔，单位为秒；必须大于 0")
	query.Flags().Int("max-poll-attempts", 0, "最多轮询次数；0 表示只查询一次")
	strictBool(query.Flags(), "poll-complete", false, "持续轮询直到任务进入终态")
	query.Flags().Float64("max-poll-timeout-seconds", 0, "轮询总时长上限，单位为秒；0 表示不限制")
	strictBool(query.Flags(), "schema", false, "输出 query-task 的 JSON Schema 描述（供 Agent 使用）")
	configureQueryTaskHelp(query)
	shared.AddCommand(query)
	return shared
}

func configureQueryTaskHelp(cmd *cobra.Command) {
	cmd.SetHelpFunc(func(helpCommand *cobra.Command, args []string) {
		helpCommand.InitDefaultHelpFlag()
		lines := []string{
			"命令: mediakit-cli shared query-task",
			"分组: shared",
			"描述: " + queryTaskDescription,
			"支持模式: Cloud",
		}
		if forceLocal {
			lines = append(
				lines,
				"",
				"错误: query-task 不支持 Local 模式，请使用 --cloud",
			)
		} else {
			lines = append(lines, "当前模式: Cloud")
		}
		lines = append(
			lines,
			"",
			"Agent 参数规则:",
			"  - "+queryTaskInputGuidance,
			"",
			"Usage:",
			"  "+helpCommand.CommandPath()+" [flags]",
		)
		if usages := strings.TrimRight(
			helpCommand.LocalFlags().FlagUsages(),
			"\n",
		); usages != "" {
			lines = append(lines, "", "Flags:", usages)
		}
		if usages := strings.TrimRight(
			helpCommand.InheritedFlags().FlagUsages(),
			"\n",
		); usages != "" {
			lines = append(lines, "", "Global Flags:", usages)
		}
		fmt.Fprintln(
			helpCommand.OutOrStdout(),
			strings.TrimRight(strings.Join(lines, "\n"), "\n"),
		)
	})
}

func queryTaskSchema() map[string]any {
	return map[string]any{
		"name":        "query_task",
		"description": queryTaskDescription,
		"input_schema": map[string]any{
			"type":        "object",
			"description": queryTaskInputGuidance,
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": taskIDDescription,
				},
				"poll_interval_seconds": map[string]any{
					"type":             "number",
					"default":          10,
					"exclusiveMinimum": 0,
					"description":      "轮询间隔，单位为秒；必须大于 0，仅在持续轮询时使用。",
				},
				"max_poll_attempts": map[string]any{
					"type":        "integer",
					"default":     0,
					"minimum":     0,
					"description": "最多轮询次数；0 表示只查询一次。",
				},
				"poll_complete": map[string]any{
					"type":        "boolean",
					"default":     false,
					"description": "是否持续轮询直到任务进入终态。",
				},
				"max_poll_timeout_seconds": map[string]any{
					"type":        "number",
					"default":     0,
					"minimum":     0,
					"description": "轮询总时长上限，单位为秒；0 表示不限制。poll_interval_seconds × max_poll_attempts 不得超过该上限。",
				},
			},
			"required": []string{"task_id"},
		},
		"output_schema": withSharedCloudResponse(map[string]any{
			"type":                 "object",
			"description":          "查询结果保留任务公共字段，并将后端 result 中的业务结果字段拍平到顶层。",
			"additionalProperties": true,
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": taskIDDescription,
				},
				"task_type": map[string]any{
					"type":        "string",
					"description": "任务类型；仅在后端实际返回非空值时出现。",
				},
				"request_id": map[string]any{
					"type":        "string",
					"description": "请求标识；仅在后端实际返回非空值时出现。",
				},
				"status": map[string]any{
					"type":        "string",
					"enum":        []string{"running", "queued", "completed", "failed", "canceled", "cancelled"},
					"description": "任务状态；completed 为成功终态，failed、canceled 或 cancelled 为失败终态。",
				},
				"success": map[string]any{
					"type":        "boolean",
					"description": "失败终态返回 false；其他状态仅在后端实际返回时出现。",
				},
				"error": map[string]any{
					"description": "失败终态的原始错误内容；仅在实际失败且后端返回时出现。",
				},
			},
		}),
	}
}

func supportCapability(support surface.SupportTool) surface.Capability {
	switch support.Name {
	case "fetch-file":
		return surface.Capability{
			Name:              support.Name,
			DisplayName:       "拉取远程文件",
			Domain:            support.Domain,
			Description:       "将 HTTP/HTTPS 远程文件拉取到本地输出目录；本地路径会被识别并直接返回。",
			Cloud:             false,
			Local:             true,
			Async:             false,
			Method:            "LOCAL",
			Path:              "",
			Parameters:        support.InputSchema,
			Response:          support.OutputSchema,
			LocalInputSchema:  support.InputSchema,
			LocalOutputSchema: support.OutputSchema,
			LocalDependencies: append([]string(nil), support.LocalDependencies...),
		}
	default:
		panic("unsupported profile support tool: " + support.Name)
	}
}

func bindCapabilityFlags(cmd *cobra.Command, capability surface.Capability) {
	properties := allCapabilityProperties(capability)
	names := sortedKeys(properties)
	for _, name := range names {
		schema := properties[name]
		flagName := strings.ReplaceAll(name, "_", "-")
		description := renderParameterFlagHelp(schema)
		defaultValue, hasDefault := schema["default"]
		switch schemaType(schema) {
		case "string":
			value := ""
			if hasDefault {
				value = fmt.Sprint(defaultValue)
			}
			cmd.Flags().String(flagName, value, description)
		case "number":
			value := 0.0
			if hasDefault {
				value, _ = numberValue(defaultValue)
			}
			cmd.Flags().Float64(flagName, value, description)
		case "integer":
			value := int64(0)
			if hasDefault {
				number, _ := numberValue(defaultValue)
				value = int64(number)
			}
			cmd.Flags().Int64(flagName, value, description)
		case "boolean":
			value := false
			if hasDefault {
				value, _ = defaultValue.(bool)
			}
			strictBool(cmd.Flags(), flagName, value, description)
		case "array":
			if isSimpleStringArray(schema) {
				value := []string(nil)
				if hasDefault {
					value = stringSlice(defaultValue)
				}
				cmd.Flags().StringSlice(flagName, value, description)
			} else {
				cmd.Flags().String(flagName, defaultJSON(defaultValue, hasDefault), description+" 使用 JSON 字符串传入。")
			}
		default:
			cmd.Flags().String(flagName, defaultJSON(defaultValue, hasDefault), description+" 使用 JSON 字符串传入。")
		}
	}
}

func collectCapabilityParams(cmd *cobra.Command, capability surface.Capability) (map[string]any, error) {
	selectedMode, selectedSchema, err := selectedCapabilityInputSchema(cmd, capability)
	if err != nil {
		return nil, err
	}
	if cmd.Flags().Changed("output-path") && selectedMode != "local" {
		return nil, fmt.Errorf(
			"--output-path 仅支持 Local 模式，请使用 --local",
		)
	}
	properties := schemaProperties(selectedSchema)
	for name := range allCapabilityProperties(capability) {
		flagName := strings.ReplaceAll(name, "_", "-")
		if cmd.Flags().Changed(flagName) {
			if _, supported := properties[name]; !supported {
				return nil, fmt.Errorf(
					"--%s 不支持当前 %s 模式",
					flagName,
					strings.Title(selectedMode),
				)
			}
		}
	}
	required := schemaStringSet(schemaStringArray(selectedSchema["required"]))
	missing := []string{}
	for name := range required {
		flagName := strings.ReplaceAll(name, "_", "-")
		if !cmd.Flags().Changed(flagName) {
			missing = append(missing, "--"+flagName)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return nil, fmt.Errorf("required flag(s) %s not set", strings.Join(missing, ", "))
	}

	params := map[string]any{}
	for _, name := range sortedKeys(properties) {
		flagName := strings.ReplaceAll(name, "_", "-")
		if !cmd.Flags().Changed(flagName) {
			continue
		}
		schema := properties[name]
		value, err := readCapabilityValue(cmd.Flags(), flagName, schema)
		if err != nil {
			return nil, fmt.Errorf("--%s 参数无效: %w", flagName, err)
		}
		if err := validateSchemaValue(schema, value, name); err != nil {
			return nil, err
		}
		params[name] = value
	}
	if err := validateObjectCombinations(selectedSchema, params, "parameters"); err != nil {
		return nil, err
	}
	return params, nil
}

func readCapabilityValue(flags *pflag.FlagSet, flagName string, schema map[string]any) (any, error) {
	switch schemaType(schema) {
	case "string":
		return flags.GetString(flagName)
	case "number":
		return flags.GetFloat64(flagName)
	case "integer":
		return flags.GetInt64(flagName)
	case "boolean":
		return flags.GetBool(flagName)
	case "array":
		if isSimpleStringArray(schema) {
			return flags.GetStringSlice(flagName)
		}
	}
	raw, err := flags.GetString(flagName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("不能为空")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("需要合法 JSON: %w", err)
	}
	return value, nil
}

func capabilityArgsValidator(capability surface.Capability) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if args[0] == "true" || args[0] == "false" {
			booleanFlags := []string{}
			for name, schema := range allCapabilityProperties(capability) {
				if schemaType(schema) == "boolean" {
					booleanFlags = append(booleanFlags, "--"+strings.ReplaceAll(name, "_", "-"))
				}
			}
			sort.Strings(booleanFlags)
			if len(booleanFlags) > 0 {
				return fmt.Errorf(
					"boolean flag 必须写成 --flag=true / --flag=false 或裸 --flag；当前 boolean 参数: %s",
					strings.Join(booleanFlags, ", "),
				)
			}
		}
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
}

func renderCapabilityHelp(
	cmd *cobra.Command,
	capability surface.Capability,
) string {
	selectedMode, selectedInput, err := selectedCapabilityInputSchema(cmd, capability)
	if err != nil {
		return strings.Join(
			[]string{
				fmt.Sprintf(
					"命令: mediakit-cli %s %s",
					capability.Domain,
					capability.Name,
				),
				fmt.Sprintf("名称: %s", capability.DisplayName),
				fmt.Sprintf("分组: %s", capability.Domain),
				fmt.Sprintf("描述: %s", capability.Description),
				"支持模式: " + supportedCapabilityModes(capability),
				"",
				"错误: " + err.Error(),
			},
			"\n",
		)
	}
	command := fmt.Sprintf(
		"mediakit-cli %s %s",
		capability.Domain,
		capability.Name,
	)
	if capability.Cloud && capability.Local && modes.LocalSurfaceVisible() {
		command = fmt.Sprintf(
			"mediakit-cli --%s %s %s",
			selectedMode,
			capability.Domain,
			capability.Name,
		)
	}
	lines := []string{
		"命令: " + command,
		fmt.Sprintf("名称: %s", capability.DisplayName),
		fmt.Sprintf("分组: %s", capability.Domain),
		fmt.Sprintf("描述: %s", capability.Description),
		"支持模式: " + supportedCapabilityModes(capability),
		"当前模式: " + strings.Title(selectedMode),
	}
	if selectedMode == "local" {
		if len(capability.LocalDependencies) > 0 {
			lines = append(
				lines,
				"本地依赖: "+strings.Join(
					append([]string(nil), capability.LocalDependencies...),
					", ",
				),
			)
		}
		lines = append(
			lines,
			"本地限制: 仅使用下方 Local 参数；Cloud-only 或当前未实现参数会明确报错，不会静默忽略。",
		)
	}
	properties := schemaProperties(selectedInput)
	if len(properties) > 0 {
		required := schemaStringSet(schemaStringArray(selectedInput["required"]))
		lines = append(lines, "", "参数:")
		for _, name := range sortedKeys(properties) {
			lines = append(
				lines,
				"  "+renderParameterDetail(
					name,
					properties[name],
					required[name],
				),
			)
		}
	}
	if guidance := strings.TrimSpace(fmt.Sprint(selectedInput["description"])); guidance != "" && guidance != "<nil>" {
		lines = append(
			lines,
			"",
			"Agent 参数规则:",
			"  - "+guidance,
		)
	}
	if selectedMode == "local" {
		lines = append(lines, "", "返回:")
		description := strings.TrimSpace(
			fmt.Sprint(capability.LocalOutputSchema["description"]),
		)
		if description != "" && description != "<nil>" {
			lines = append(lines, "  - "+description)
		}
		for _, name := range sortedKeys(
			schemaProperties(capability.LocalOutputSchema),
		) {
			field := schemaProperties(capability.LocalOutputSchema)[name]
			fieldDescription := strings.TrimSpace(
				fmt.Sprint(field["description"]),
			)
			if fieldDescription == "" || fieldDescription == "<nil>" {
				lines = append(lines, "  - "+name)
				continue
			}
			lines = append(
				lines,
				fmt.Sprintf("  - %s: %s", name, fieldDescription),
			)
		}
	} else if capability.Async {
		lines = append(
			lines,
			"",
			"返回:",
			"  - task_id: "+taskIDDescription,
			"  - task_type: 后端实际返回非空值时透传；未返回时不生成该字段。",
			"  - 使用 `mediakit-cli shared query-task --task-id <task_id>` 查询任务状态并获取最终结果。",
		)
	} else {
		lines = append(
			lines,
			"",
			"返回:",
			"  - 同步返回业务结果；字段以 `--schema` 的 output_schema 为准。",
		)
	}
	return strings.Join(lines, "\n")
}

func supportedCapabilityModes(capability surface.Capability) string {
	supported := []string{}
	if capability.Cloud {
		supported = append(supported, "Cloud")
	}
	if capability.Local && modes.LocalSurfaceVisible() {
		supported = append(supported, "Local")
	}
	if len(supported) == 0 {
		return "无"
	}
	return strings.Join(supported, " + ")
}

func renderParameterDetail(
	name string,
	schema map[string]any,
	required bool,
) string {
	parts := []string{schemaType(schema)}
	if required {
		parts = append(parts, "required")
	}
	if value, ok := schema["default"]; ok {
		parts = append(parts, "default="+compactJSON(value))
	}
	line := fmt.Sprintf(
		"- --%s [%s]",
		strings.ReplaceAll(name, "_", "-"),
		strings.Join(parts, ", "),
	)
	if description := strings.TrimSpace(fmt.Sprint(schema["description"])); description != "" && description != "<nil>" {
		line += " " + description
	}
	if values := schemaArray(schema["enum"]); len(values) > 0 {
		line += " 可选值: " + joinJSONValues(values)
	}
	line += renderNestedSchema(schema, " ")
	switch {
	case isSimpleStringArray(schema):
		line += " CLI 传参时可使用逗号分隔多个值，或重复传递该 flag；不要传 JSON 数组字符串。"
	case schemaType(schema) == "object",
		schemaType(schema) == "array":
		line += " CLI 传参时必须使用合法 JSON 字符串。"
	}
	return line
}

func selectedCapabilityInputSchema(
	cmd *cobra.Command,
	capability surface.Capability,
) (string, map[string]any, error) {
	selectedMode := "cloud"
	if cmd == nil {
		if !capability.Cloud && capability.Local {
			selectedMode = "local"
		}
	} else {
		mode, err := modes.SchemaMode(cmd, capabilityRuntimeMeta(capability))
		if err != nil {
			return "", nil, err
		}
		selectedMode = mode
	}
	if selectedMode == "local" {
		if !capability.Local || len(capability.LocalInputSchema) == 0 {
			return "", nil, fmt.Errorf("%s 不支持 Local 模式", capability.Name)
		}
		return selectedMode, capability.LocalInputSchema, nil
	}
	if !capability.Cloud || len(capability.Parameters) == 0 {
		return "", nil, fmt.Errorf("%s 不支持 Cloud 模式", capability.Name)
	}
	return selectedMode, capability.Parameters, nil
}

func allCapabilityProperties(
	capability surface.Capability,
) map[string]map[string]any {
	merged := map[string]map[string]any{}
	for name, schema := range schemaProperties(capability.Parameters) {
		merged[name] = schema
	}
	for name, schema := range schemaProperties(capability.LocalInputSchema) {
		if _, exists := merged[name]; !exists {
			merged[name] = schema
		}
	}
	return merged
}

func capabilityRuntimeMeta(capability surface.Capability) modes.CapabilityRuntimeMeta {
	return modes.CapabilityRuntimeMeta{
		Name:           capability.Name,
		Domain:         capability.Domain,
		Description:    capability.Description,
		CloudOnly:      !capability.Local,
		LocalSupported: capability.Local,
		LocalDeps:      append([]string(nil), capability.LocalDependencies...),
	}
}

func localUnsupportedParams(
	capability surface.Capability,
	params map[string]any,
) []string {
	localProperties := schemaProperties(capability.LocalInputSchema)
	unsupported := []string{}
	for name := range params {
		if _, supported := localProperties[name]; !supported {
			unsupported = append(
				unsupported,
				"--"+strings.ReplaceAll(name, "_", "-"),
			)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func renderParameterFlagHelp(schema map[string]any) string {
	description := strings.TrimSpace(fmt.Sprint(schema["description"]))
	if description == "<nil>" {
		description = ""
	}
	if items, ok := schema["items"].(map[string]any); ok {
		itemDescription := strings.TrimSpace(fmt.Sprint(items["description"]))
		if itemDescription != "" &&
			itemDescription != "<nil>" &&
			!strings.Contains(description, itemDescription) {
			if description != "" {
				description += " "
			}
			description += "子项说明：" + itemDescription
		}
		if values := schemaArray(items["enum"]); len(values) > 0 {
			if description != "" {
				description += " "
			}
			description += "子项可选值: " + joinJSONValues(values)
		}
	}
	if isSimpleStringArray(schema) {
		if description != "" {
			description += " "
		}
		description += "CLI 传参时可使用逗号分隔多个值，或重复传递该 flag；不要传 JSON 数组字符串。"
	}
	if values := schemaArray(schema["enum"]); len(values) > 0 {
		if description != "" {
			description += " "
		}
		description += "可选值: " + joinJSONValues(values)
	}
	return description
}

func renderNestedSchema(schema map[string]any, prefix string) string {
	switch schemaType(schema) {
	case "object":
		properties := schemaProperties(schema)
		if len(properties) == 0 {
			return ""
		}
		required := schemaStringSet(schemaStringArray(schema["required"]))
		parts := []string{" 子字段:"}
		for _, name := range sortedKeys(properties) {
			label := name
			if required[name] {
				label += "(必填)"
			}
			description := strings.TrimSpace(fmt.Sprint(properties[name]["description"]))
			if description == "<nil>" {
				description = ""
			}
			parts = append(parts, fmt.Sprintf("%s%s: %s", prefix, label, description))
		}
		return strings.Join(parts, prefix)
	case "array":
		items, _ := schema["items"].(map[string]any)
		parts := []string{}
		itemDescription := strings.TrimSpace(fmt.Sprint(items["description"]))
		if itemDescription != "" && itemDescription != "<nil>" {
			parts = append(parts, " 子项说明："+itemDescription)
		}
		if schemaType(items) == "object" {
			parts = append(parts, renderNestedSchema(items, prefix))
		}
		if values := schemaArray(items["enum"]); len(values) > 0 {
			parts = append(parts, " 子项可选值: "+joinJSONValues(values))
		}
		return strings.Join(parts, "")
	}
	return ""
}

func configureCapabilityHelp(cmd *cobra.Command, capability surface.Capability) {
	businessFlags := map[string]struct{}{}
	for name := range allCapabilityProperties(capability) {
		businessFlags[strings.ReplaceAll(name, "_", "-")] = struct{}{}
	}
	cmd.SetHelpFunc(func(helpCommand *cobra.Command, args []string) {
		helpCommand.InitDefaultHelpFlag()
		excludedFlags := map[string]struct{}{}
		for name := range businessFlags {
			excludedFlags[name] = struct{}{}
		}
		selectedMode, _, err := selectedCapabilityInputSchema(
			helpCommand,
			capability,
		)
		if err == nil && selectedMode != "local" {
			excludedFlags["output-path"] = struct{}{}
		}
		lines := []string{
			strings.TrimRight(
				renderCapabilityHelp(helpCommand, capability),
				"\n",
			),
			"",
			"Usage:",
			"  " + helpCommand.UseLine(),
		}
		if usages := filteredFlagUsages(helpCommand.LocalFlags(), excludedFlags); usages != "" {
			lines = append(lines, "", "Flags:", usages)
		}
		if usages := strings.TrimRight(helpCommand.InheritedFlags().FlagUsages(), "\n"); usages != "" {
			lines = append(lines, "", "Global Flags:", usages)
		}
		fmt.Fprintln(helpCommand.OutOrStdout(), strings.TrimRight(strings.Join(lines, "\n"), "\n"))
	})
}

func filteredFlagUsages(source *pflag.FlagSet, excluded map[string]struct{}) string {
	flags := pflag.NewFlagSet(source.Name(), pflag.ContinueOnError)
	flags.SortFlags = source.SortFlags
	source.VisitAll(func(flag *pflag.Flag) {
		if _, skip := excluded[flag.Name]; skip {
			return
		}
		flags.AddFlag(flag)
	})
	if !flags.HasAvailableFlags() {
		return ""
	}
	return strings.TrimRight(flags.FlagUsages(), "\n")
}

func buildCapabilitySchema(
	cmd *cobra.Command,
	capability surface.Capability,
) (map[string]any, error) {
	selectedMode, inputSchema, err := selectedCapabilityInputSchema(cmd, capability)
	if err != nil {
		return nil, err
	}
	var outputSchema map[string]any
	if selectedMode == "local" {
		outputSchema = capability.LocalOutputSchema
	} else {
		outputSchema = cloudOutputSchema(capability)
	}
	if len(outputSchema) == 0 {
		return nil, fmt.Errorf(
			"%s 不支持 %s 模式",
			capability.Name,
			strings.Title(selectedMode),
		)
	}
	return map[string]any{
		"name":          strings.ReplaceAll(capability.Name, "-", "_"),
		"description":   capability.Description,
		"input_schema":  inputSchema,
		"output_schema": outputSchema,
	}, nil
}

func cloudOutputSchema(capability surface.Capability) map[string]any {
	if capability.Async {
		return map[string]any{
			"type":        "object",
			"description": "Cloud 异步调用的提交结果。",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"minLength":   1,
					"description": taskIDDescription,
				},
				"task_type": map[string]any{
					"type":        "string",
					"description": "任务类型；仅在后端实际返回非空值时出现。",
				},
				"request_id": map[string]any{
					"type":        "string",
					"description": "请求标识；仅在后端返回时出现。",
				},
			},
			"required": []string{"task_id"},
		}
	}
	return withSharedCloudResponse(capability.Response)
}

func withSharedCloudResponse(base map[string]any) map[string]any {
	baseProperties, ok := base["properties"].(map[string]any)
	if !ok {
		panic("Cloud response schema properties must be an object")
	}
	shared := surface.Current().SharedCloudResponseSchema
	sharedProperties, ok := shared["properties"].(map[string]any)
	if !ok {
		panic("shared Cloud response schema properties must be an object")
	}

	merged := make(map[string]any, len(base)+1)
	for key, value := range base {
		merged[key] = value
	}
	properties := make(map[string]any, len(baseProperties)+len(sharedProperties))
	for key, value := range baseProperties {
		properties[key] = value
	}
	for key, value := range sharedProperties {
		if _, exists := properties[key]; exists {
			panic("shared Cloud response field collides with capability response: " + key)
		}
		properties[key] = value
	}
	merged["properties"] = properties
	return merged
}

func validateSchemaValue(schema map[string]any, value any, path string) error {
	if !matchesType(schemaType(schema), value) {
		return fmt.Errorf("%s 类型必须为 %s", path, schemaType(schema))
	}
	if constant, ok := schema["const"]; ok && !equalJSONScalar(constant, value) {
		return fmt.Errorf("%s 必须等于 %s", path, compactJSON(constant))
	}
	if values := schemaArray(schema["enum"]); len(values) > 0 {
		matched := false
		for _, allowed := range values {
			if equalJSONScalar(allowed, value) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s 必须是以下值之一: %s", path, joinJSONValues(values))
		}
	}
	switch typed := value.(type) {
	case string:
		if minimum, ok := integerConstraint(schema["minLength"]); ok && len([]rune(typed)) < minimum {
			return fmt.Errorf("%s 长度不能小于 %d", path, minimum)
		}
		if maximum, ok := integerConstraint(schema["maxLength"]); ok && len([]rune(typed)) > maximum {
			return fmt.Errorf("%s 长度不能大于 %d", path, maximum)
		}
		if pattern, ok := schema["pattern"].(string); ok && pattern != "" {
			expression, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("%s 的 Schema pattern 无效: %w", path, err)
			}
			if !expression.MatchString(typed) {
				return fmt.Errorf("%s 不符合格式约束 %s", path, pattern)
			}
		}
	case []string:
		values := make([]any, len(typed))
		for index := range typed {
			values[index] = typed[index]
		}
		if err := validateArray(schema, values, path); err != nil {
			return err
		}
	case []any:
		if err := validateArray(schema, typed, path); err != nil {
			return err
		}
	case map[string]any:
		if err := validateObject(schema, typed, path); err != nil {
			return err
		}
	default:
		if number, ok := numberValue(value); ok {
			if minimum, ok := numberConstraint(schema["minimum"]); ok && number < minimum {
				return fmt.Errorf("%s 不能小于 %s", path, compactJSON(schema["minimum"]))
			}
			if maximum, ok := numberConstraint(schema["maximum"]); ok && number > maximum {
				return fmt.Errorf("%s 不能大于 %s", path, compactJSON(schema["maximum"]))
			}
			if minimum, ok := numberConstraint(schema["exclusiveMinimum"]); ok && number <= minimum {
				return fmt.Errorf("%s 必须大于 %s", path, compactJSON(schema["exclusiveMinimum"]))
			}
			if maximum, ok := numberConstraint(schema["exclusiveMaximum"]); ok && number >= maximum {
				return fmt.Errorf("%s 必须小于 %s", path, compactJSON(schema["exclusiveMaximum"]))
			}
			if multiple, ok := numberConstraint(schema["multipleOf"]); ok && multiple != 0 {
				quotient := number / multiple
				if quotient != float64(int64(quotient)) {
					return fmt.Errorf("%s 必须是 %s 的整数倍", path, compactJSON(schema["multipleOf"]))
				}
			}
		}
	}
	return validateCombinators(schema, value, path)
}

func validateArray(schema map[string]any, values []any, path string) error {
	if minimum, ok := integerConstraint(schema["minItems"]); ok && len(values) < minimum {
		return fmt.Errorf("%s 至少需要 %d 项", path, minimum)
	}
	if maximum, ok := integerConstraint(schema["maxItems"]); ok && len(values) > maximum {
		return fmt.Errorf("%s 最多允许 %d 项", path, maximum)
	}
	if unique, _ := schema["uniqueItems"].(bool); unique {
		for left := range values {
			for right := left + 1; right < len(values); right++ {
				if reflect.DeepEqual(values[left], values[right]) {
					return fmt.Errorf("%s 不允许重复项", path)
				}
			}
		}
	}
	items, _ := schema["items"].(map[string]any)
	for index, value := range values {
		if len(items) > 0 {
			if err := validateSchemaValue(items, value, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateObject(schema map[string]any, value map[string]any, path string) error {
	properties := schemaProperties(schema)
	if minimum, ok := integerConstraint(schema["minProperties"]); ok && len(value) < minimum {
		return fmt.Errorf("%s 至少需要 %d 个字段", path, minimum)
	}
	if maximum, ok := integerConstraint(schema["maxProperties"]); ok && len(value) > maximum {
		return fmt.Errorf("%s 最多允许 %d 个字段", path, maximum)
	}
	for _, required := range schemaStringArray(schema["required"]) {
		if _, ok := value[required]; !ok {
			return fmt.Errorf("%s.%s 为必填字段", path, required)
		}
	}
	for name, child := range value {
		childSchema, ok := properties[name]
		if !ok {
			if additional, exists := schema["additionalProperties"]; exists {
				if additional == false {
					return fmt.Errorf("%s.%s 不是允许的字段", path, name)
				}
				if additionalSchema, ok := additional.(map[string]any); ok {
					if err := validateSchemaValue(additionalSchema, child, path+"."+name); err != nil {
						return err
					}
				}
			}
			continue
		}
		if err := validateSchemaValue(childSchema, child, path+"."+name); err != nil {
			return err
		}
	}
	return nil
}

func validateObjectCombinations(schema map[string]any, value map[string]any, path string) error {
	return validateCombinators(schema, value, path)
}

func validateCombinators(schema map[string]any, value any, path string) error {
	if allOf := schemaArray(schema["allOf"]); len(allOf) > 0 {
		for _, item := range allOf {
			child, ok := item.(map[string]any)
			if ok {
				if err := validateSchemaValue(child, value, path); err != nil {
					return err
				}
			}
		}
	}
	for keyword, exact := range map[string]bool{"anyOf": false, "oneOf": true} {
		options := schemaArray(schema[keyword])
		if len(options) == 0 {
			continue
		}
		matches := 0
		for _, item := range options {
			child, ok := item.(map[string]any)
			if ok && validateSchemaValue(child, value, path) == nil {
				matches++
			}
		}
		if (exact && matches != 1) || (!exact && matches == 0) {
			return fmt.Errorf("%s 不满足 %s 约束", path, keyword)
		}
	}
	if condition, ok := schema["if"].(map[string]any); ok {
		branch := "else"
		if validateSchemaValue(condition, value, path) == nil {
			branch = "then"
		}
		if child, ok := schema[branch].(map[string]any); ok {
			if err := validateSchemaValue(child, value, path); err != nil {
				return err
			}
		}
	}
	if child, ok := schema["not"].(map[string]any); ok &&
		validateSchemaValue(child, value, path) == nil {
		return fmt.Errorf("%s 命中禁止的参数组合", path)
	}
	return nil
}

func writeCapabilityError(writer io.Writer, err error) error {
	payload, ok := localDependencyErrorPayload(err)
	if !ok {
		payload = map[string]any{
			"error": map[string]any{
				"type":    classifyErrorType(err),
				"code":    classifyErrorCode(err),
				"message": err.Error(),
			},
		}
	}
	notice.Inject(payload)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(payload); encodeErr != nil {
		return encodeErr
	}
	return cliexit.ErrBusinessFailure
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func classifyErrorType(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "禁止") ||
		strings.Contains(message, "不在白名单") ||
		strings.Contains(message, "不安全字符"):
		return "SecurityViolation"
	case strings.Contains(message, "本地处理器未实现") ||
		strings.Contains(message, "不支持本地执行") ||
		strings.Contains(message, "不支持 Local 模式") ||
		strings.Contains(message, "不支持当前 Local 模式") ||
		strings.Contains(message, "已禁用 Local 模式") ||
		strings.Contains(message, "Local 模式已禁用") ||
		strings.Contains(message, "本地依赖"):
		return "EnvironmentError"
	case strings.Contains(message, "未配置云端鉴权") ||
		strings.Contains(message, "云端鉴权不可用"):
		return "AuthenticationError"
	case strings.Contains(message, "required flag(s)") ||
		strings.Contains(message, "必填参数") ||
		strings.Contains(message, "为必填字段") ||
		strings.Contains(message, "不能为空") ||
		strings.Contains(message, "必须是") ||
		strings.Contains(message, "类型必须为") ||
		strings.Contains(message, "需要合法 JSON") ||
		strings.Contains(message, "取值范围") ||
		strings.Contains(message, "不能小于") ||
		strings.Contains(message, "不能大于") ||
		strings.Contains(message, "长度不能") ||
		strings.Contains(message, "至少需要") ||
		strings.Contains(message, "最多允许") ||
		strings.Contains(message, "仅支持") ||
		strings.Contains(message, "必须大于") ||
		strings.Contains(message, "必须小于"):
		return "InvalidParameter"
	default:
		return "ExecutionError"
	}
}

func classifyErrorCode(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "禁止"):
		return "ForbiddenOperation"
	case strings.Contains(message, "不在白名单"):
		return "NotWhitelisted"
	case strings.Contains(message, "不安全字符"):
		return "UnsafeCharacters"
	case strings.Contains(message, "本地处理器未实现"):
		return "HandlerNotImplemented"
	case strings.Contains(message, "不支持本地执行") ||
		strings.Contains(message, "不支持 Local 模式") ||
		strings.Contains(message, "不支持当前 Local 模式") ||
		strings.Contains(message, "已禁用 Local 模式") ||
		strings.Contains(message, "Local 模式已禁用"):
		return "LocalUnsupported"
	case strings.Contains(message, "本地依赖"):
		return "DependencyMissing"
	case strings.Contains(message, "未配置云端鉴权") ||
		strings.Contains(message, "云端鉴权不可用"):
		return "MissingAuthentication"
	case strings.Contains(message, "required flag(s)") ||
		strings.Contains(message, "必填参数") ||
		strings.Contains(message, "为必填字段") ||
		strings.Contains(message, "不能为空"):
		return "MissingRequiredParam"
	case strings.Contains(message, "类型必须为") ||
		strings.Contains(message, "必须是字符串数组") ||
		strings.Contains(message, "需要合法 JSON"):
		return "InvalidParamType"
	case strings.Contains(message, "必须大于等于") ||
		strings.Contains(message, "必须大于") ||
		strings.Contains(message, "必须小于") ||
		strings.Contains(message, "不能小于") ||
		strings.Contains(message, "不能大于") ||
		strings.Contains(message, "长度不能") ||
		strings.Contains(message, "取值范围") ||
		strings.Contains(message, "最多允许"):
		return "ParamOutOfRange"
	case strings.Contains(message, "至少需要"):
		return "ParamInsufficient"
	case strings.Contains(message, "必须是以下值之一") ||
		strings.Contains(message, "仅支持"):
		return "UnsupportedValue"
	case strings.Contains(message, "download failed"):
		return "DownloadFailed"
	case strings.Contains(message, "执行失败"):
		return "ExecutionFailed"
	default:
		return "Unknown"
	}
}

func schemaProperties(schema map[string]any) map[string]map[string]any {
	result := map[string]map[string]any{}
	raw, _ := schema["properties"].(map[string]any)
	for name, value := range raw {
		if child, ok := value.(map[string]any); ok {
			result[name] = child
		}
	}
	return result
}

func schemaType(schema map[string]any) string {
	value, _ := schema["type"].(string)
	if value != "" {
		return value
	}
	if _, ok := schema["properties"]; ok {
		return "object"
	}
	if _, ok := schema["required"]; ok {
		return "object"
	}
	if _, ok := schema["items"]; ok {
		return "array"
	}
	return ""
}

func schemaArray(value any) []any {
	values, _ := value.([]any)
	return values
}

func schemaStringArray(value any) []string {
	raw := schemaArray(value)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func sortedKeys(values map[string]map[string]any) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func schemaStringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func isSimpleStringArray(schema map[string]any) bool {
	items, _ := schema["items"].(map[string]any)
	return schemaType(items) == "string"
}

func defaultJSON(value any, present bool) string {
	if !present {
		return ""
	}
	return compactJSON(value)
}

func compactJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func joinJSONValues(values []any) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = compactJSON(value)
	}
	return strings.Join(parts, ", ")
}

func stringSlice(value any) []string {
	raw := schemaArray(value)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		result = append(result, fmt.Sprint(item))
	}
	return result
}

func matchesType(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := numberValue(value)
		return ok
	case "integer":
		number, ok := numberValue(value)
		return ok && number == float64(int64(number))
	case "array":
		switch value.(type) {
		case []any, []string:
			return true
		}
		return false
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return true
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		number, err := strconv.ParseFloat(string(typed), 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func numberConstraint(value any) (float64, bool) {
	return numberValue(value)
}

func integerConstraint(value any) (int, bool) {
	number, ok := numberValue(value)
	return int(number), ok
}

func equalJSONScalar(left any, right any) bool {
	leftNumber, leftIsNumber := numberValue(left)
	rightNumber, rightIsNumber := numberValue(right)
	if leftIsNumber || rightIsNumber {
		return leftIsNumber && rightIsNumber && leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}
