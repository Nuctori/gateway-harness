package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/steward"
)

type ConfigAssistantRequest struct {
	Message string                 `json:"message"`
	Config  adapter.GoalGateConfig `json:"config"`
}

type ConfigAssistantChange struct {
	Path   string `json:"path"`
	From   any    `json:"from,omitempty"`
	To     any    `json:"to,omitempty"`
	Reason string `json:"reason"`
}

type ConfigAssistantResponse struct {
	AssistantMessage     string                  `json:"assistant_message"`
	Proposal             adapter.GoalGateConfig  `json:"proposal"`
	Changes              []ConfigAssistantChange `json:"changes,omitempty"`
	Warnings             []string                `json:"warnings,omitempty"`
	RequiresConfirmation bool                    `json:"requires_confirmation"`
}

var (
	maxContinueAttemptsPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)max[_ ]?continue[_ ]?attempts\D*(\d+)`),
		regexp.MustCompile(`最多(?:继续|重试|续跑)?\D*(\d+)`),
	}
	cooldownSecondsPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)cooldown(?:[_ ]?seconds)?\D*(\d+)`),
		regexp.MustCompile(`(?:冷却|间隔)\D*(\d+)\s*(?:秒|s|seconds?)?`),
	}
)

func ComposeGoalGateConfigDraft(current adapter.GoalGateConfig, message string) ConfigAssistantResponse {
	proposal := cloneGoalGateConfig(current)
	trimmedMessage := strings.TrimSpace(message)
	normalized := strings.ToLower(trimmedMessage)
	var changes []ConfigAssistantChange
	var warnings []string

	if strings.TrimSpace(proposal.Hook) == "" {
		proposal.Hook = steward.GoalBeforeCompleteHook
		changes = append(changes, ConfigAssistantChange{
			Path:   "hook",
			From:   "",
			To:     steward.GoalBeforeCompleteHook,
			Reason: "Goal Gate currently supports explicit completion review on goal.before_complete.",
		})
	}

	if mentionsAny(normalized, "disable", "disabled", "turn off", "关闭", "停用", "禁用") {
		if proposal.Enabled {
			changes = append(changes, ConfigAssistantChange{Path: "enabled", From: true, To: false, Reason: "The request asked to disable Goal Gate explicitly."})
		}
		proposal.Enabled = false
	} else if mentionsAny(normalized, "enable", "enabled", "turn on", "开启", "启用", "配置", "setup", "review", "goal gate", "审查") {
		if !proposal.Enabled {
			changes = append(changes, ConfigAssistantChange{Path: "enabled", From: false, To: true, Reason: "The request asked for an explicit Goal Gate review flow."})
		}
		proposal.Enabled = true
	}

	if mentionsAny(normalized, "smolagents", "python", "agent") {
		changes = appendConfigChange(changes, &proposal.Runner.Command, "runner.command", "python", "Use the lightweight Python smolagents example runner as the external reviewer.")
		changes = appendConfigChange(changes, &proposal.Runner.Workdir, "runner.workdir", "../..", "Resolve the example runner from the repository root.")
		changes = appendSliceConfigChange(changes, &proposal.Runner.Args, "runner.args", []string{"examples/smolagents/goal_reviewer.py"}, "Launch the bundled goal reviewer example.")
	} else if mentionsAny(normalized, "helper", "demo helper", "go helper", "test helper") {
		changes = appendConfigChange(changes, &proposal.Runner.Command, "runner.command", "go", "Use the local Go helper for deterministic smoke testing.")
		changes = appendConfigChange(changes, &proposal.Runner.Workdir, "runner.workdir", "../..", "Resolve the helper from the repository root.")
		changes = appendSliceConfigChange(changes, &proposal.Runner.Args, "runner.args", []string{"run", "./cmd/gateway-harness/testdata/goalreviewhelper"}, "Launch the bundled deterministic goal review helper.")
	} else if proposal.Enabled && strings.TrimSpace(proposal.Runner.Command) == "" {
		changes = appendConfigChange(changes, &proposal.Runner.Command, "runner.command", "python", "Default to the documented external runner example when enabling Goal Gate from chat.")
		changes = appendConfigChange(changes, &proposal.Runner.Workdir, "runner.workdir", "../..", "Resolve the example runner from the repository root.")
		changes = appendSliceConfigChange(changes, &proposal.Runner.Args, "runner.args", []string{"examples/smolagents/goal_reviewer.py"}, "Launch the bundled goal reviewer example.")
	}

	if proposal.Enabled {
		changes = ensureGoalReviewDefaults(changes, &proposal)
	}
	if proposal.Enabled {
		changes = ensureExplicitlyAuthorizedExtendedInputs(changes, &proposal, normalized)
	}
	if mentionsAny(normalized, "before resume", "resume hook", "恢复前", "继续前") {
		warnings = append(warnings, "The current Goal Gate host keeps the hook selector on goal.before_complete. goal.before_resume is available in the broader hook catalog, but this host does not expose it as a runnable Goal Gate setting yet.")
	}
	if mentionsAny(normalized, "after complete", "post complete", "完成后", "收尾后") {
		warnings = append(warnings, "The broader hook catalog includes goal.after_complete for post-completion observation, but the current Goal Gate host still limits runnable interception to goal.before_complete.")
	}

	if mentionsAny(normalized, "inject", "context.inject", "注入", "补充上下文") {
		changes = ensureStringSliceMember(changes, &proposal.AllowedActions, "allowed_actions", "context.inject", "Allow the reviewer to propose explicit continuation patches instead of hidden mutation.")
	}

	if attempts, ok := extractNumber(maxContinueAttemptsPatterns, trimmedMessage); ok {
		changes = appendIntConfigChange(changes, &proposal.MaxContinueAttempts, "max_continue_attempts", attempts, "Apply the requested explicit continuation limit.")
	}
	if cooldown, ok := extractNumber(cooldownSecondsPatterns, trimmedMessage); ok {
		changes = appendIntConfigChange(changes, &proposal.CooldownSeconds, "cooldown_seconds", cooldown, "Apply the requested explicit cooldown window.")
	}

	if trimmedMessage == "" {
		warnings = append(warnings, "No message was provided, so the assistant returned the current explicit config draft unchanged.")
	}
	if !proposal.Enabled {
		warnings = append(warnings, "Goal Gate remains disabled until you explicitly apply and save a draft that enables it.")
	}
	if strings.TrimSpace(proposal.Runner.Command) == "" {
		warnings = append(warnings, "No external runner is configured yet. The host still needs an explicit agent command before Goal Gate can execute.")
	}

	messageText := buildAssistantReply(proposal, changes, warnings)
	return ConfigAssistantResponse{
		AssistantMessage:     messageText,
		Proposal:             proposal,
		Changes:              changes,
		Warnings:             warnings,
		RequiresConfirmation: true,
	}
}

func buildAssistantReply(proposal adapter.GoalGateConfig, changes []ConfigAssistantChange, warnings []string) string {
	status := "disabled"
	if proposal.Enabled {
		status = "enabled"
	}
	parts := []string{
		fmt.Sprintf("I generated an explicit Goal Gate draft and kept it proposal-only. Current draft status: %s on %s.", status, proposal.Hook),
	}
	if len(changes) > 0 {
		parts = append(parts, fmt.Sprintf("The draft updates %d explicit field(s), so you can review or apply them one by one from the UI.", len(changes)))
	} else {
		parts = append(parts, "I did not detect a concrete config change request, so the draft stays aligned with the current visible settings.")
	}
	if len(warnings) > 0 {
		parts = append(parts, warnings[0])
	}
	parts = append(parts, "Nothing is auto-applied: saving the config is still an explicit host action.")
	return strings.Join(parts, " ")
}

func ensureGoalReviewDefaults(changes []ConfigAssistantChange, proposal *adapter.GoalGateConfig) []ConfigAssistantChange {
	proposal.Hook = steward.GoalBeforeCompleteHook
	changes = ensureStringSliceMembers(changes, &proposal.AllowedInputs, "allowed_inputs", defaultAllowedInputs(), "Give the reviewer only the redacted goal-review inputs supported by the steward spec.")
	changes = ensureStringSliceMembers(changes, &proposal.AllowedActions, "allowed_actions", defaultAllowedActions(), "Allow only explicit review outcomes and audit-safe artifacts by default.")
	if proposal.MaxContinueAttempts == 0 {
		changes = appendIntConfigChange(changes, &proposal.MaxContinueAttempts, "max_continue_attempts", 3, "Use the documented default retry ceiling for explicit continue loops.")
	}
	if proposal.CooldownSeconds == 0 {
		changes = appendIntConfigChange(changes, &proposal.CooldownSeconds, "cooldown_seconds", 60, "Use the documented default cooldown for repeated continuation attempts.")
	}
	return changes
}

func defaultAllowedInputs() []string {
	return []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace"}
}

func ensureExplicitlyAuthorizedExtendedInputs(changes []ConfigAssistantChange, proposal *adapter.GoalGateConfig, normalizedMessage string) []ConfigAssistantChange {
	if authorizesFullGoalContext(normalizedMessage) {
		return ensureStringSliceMembers(changes, &proposal.AllowedInputs, "allowed_inputs", extendedAllowedInputs(), "The request explicitly authorized a broader redacted review scope.")
	}
	if mentionsAny(normalizedMessage, "changed files", "changed_files", "变更文件", "改动文件") {
		changes = ensureStringSliceMember(changes, &proposal.AllowedInputs, "allowed_inputs", "changed_files", "The request explicitly authorized changed file summaries.")
	}
	if mentionsAny(normalizedMessage, "verification summary", "verification_summary", "验证摘要", "部署摘要") {
		changes = ensureStringSliceMember(changes, &proposal.AllowedInputs, "allowed_inputs", "verification_summary", "The request explicitly authorized broader verification status.")
	}
	if mentionsAny(normalizedMessage, "user goal", "user_goal", "用户目标", "原始目标") {
		changes = ensureStringSliceMember(changes, &proposal.AllowedInputs, "allowed_inputs", "user_goal", "The request explicitly authorized forwarding the redacted user objective.")
	}
	return changes
}

func extendedAllowedInputs() []string {
	return []string{"changed_files", "verification_summary", "user_goal"}
}

func defaultAllowedActions() []string {
	return []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"}
}

func authorizesFullGoalContext(text string) bool {
	return mentionsAny(text,
		"full context",
		"broader context",
		"more context",
		"complete context",
		"完整上下文",
		"更完整上下文",
		"更多上下文",
		"全部上下文",
		"全部摘要",
	)
}

func mentionsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func extractNumber(patterns []*regexp.Regexp, text string) (int, bool) {
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) != 2 {
			continue
		}
		var value int
		_, err := fmt.Sscanf(matches[1], "%d", &value)
		if err == nil && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func cloneGoalGateConfig(in adapter.GoalGateConfig) adapter.GoalGateConfig {
	out := in
	out.AllowedInputs = append([]string(nil), in.AllowedInputs...)
	out.AllowedActions = append([]string(nil), in.AllowedActions...)
	out.Runner.Args = append([]string(nil), in.Runner.Args...)
	return out
}

func appendConfigChange(changes []ConfigAssistantChange, target *string, path string, value string, reason string) []ConfigAssistantChange {
	if *target == value {
		return changes
	}
	changes = append(changes, ConfigAssistantChange{Path: path, From: *target, To: value, Reason: reason})
	*target = value
	return changes
}

func appendIntConfigChange(changes []ConfigAssistantChange, target *int, path string, value int, reason string) []ConfigAssistantChange {
	if *target == value {
		return changes
	}
	changes = append(changes, ConfigAssistantChange{Path: path, From: *target, To: value, Reason: reason})
	*target = value
	return changes
}

func appendSliceConfigChange(changes []ConfigAssistantChange, target *[]string, path string, value []string, reason string) []ConfigAssistantChange {
	if slicesEqual(*target, value) {
		return changes
	}
	from := append([]string(nil), (*target)...)
	to := append([]string(nil), value...)
	changes = append(changes, ConfigAssistantChange{Path: path, From: from, To: to, Reason: reason})
	*target = to
	return changes
}

func ensureStringSliceMembers(changes []ConfigAssistantChange, target *[]string, path string, values []string, reason string) []ConfigAssistantChange {
	for _, value := range values {
		changes = ensureStringSliceMember(changes, target, path, value, reason)
	}
	return changes
}

func ensureStringSliceMember(changes []ConfigAssistantChange, target *[]string, path string, value string, reason string) []ConfigAssistantChange {
	for _, existing := range *target {
		if existing == value {
			return changes
		}
	}
	before := append([]string(nil), (*target)...)
	*target = append(*target, value)
	changes = append(changes, ConfigAssistantChange{Path: path, From: before, To: append([]string(nil), (*target)...), Reason: reason})
	return changes
}

func slicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
