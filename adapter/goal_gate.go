package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/steward"
)

const GoalGateSkipDisabled = "goal_gate_disabled"

func DecodeGoalGateConfig(r io.Reader) (GoalGateConfig, error) {
	var cfg GoalGateConfig
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return GoalGateConfig{}, err
	}
	return cfg, nil
}

func ValidateGoalGateConfig(cfg GoalGateConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Hook) != steward.GoalBeforeCompleteHook {
		return fmt.Errorf("goal gate hook must be %q", steward.GoalBeforeCompleteHook)
	}
	if strings.TrimSpace(cfg.Runner.Command) == "" {
		return fmt.Errorf("goal gate runner.command is required when enabled")
	}
	if len(cfg.AllowedInputs) == 0 {
		return fmt.Errorf("goal gate allowed_inputs is required when enabled")
	}
	if len(cfg.AllowedActions) == 0 {
		return fmt.Errorf("goal gate allowed_actions is required when enabled")
	}
	if !containsValue(cfg.AllowedInputs, "goal_state") {
		return fmt.Errorf("goal gate allowed_inputs must include %q", "goal_state")
	}
	for _, action := range []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue"} {
		if !containsValue(cfg.AllowedActions, action) {
			return fmt.Errorf("goal gate allowed_actions must include %q", action)
		}
	}
	if cfg.MaxContinueAttempts < 0 {
		return fmt.Errorf("goal gate max_continue_attempts must be non-negative")
	}
	if cfg.CooldownSeconds < 0 {
		return fmt.Errorf("goal gate cooldown_seconds must be non-negative")
	}
	if err := validateGoalGateInputs(cfg.AllowedInputs); err != nil {
		return err
	}
	if err := validateGoalGateActions(cfg.AllowedActions); err != nil {
		return err
	}
	return nil
}

func ExecuteGoalGate(ctx context.Context, req GoalGateRequest) (GoalGateResult, error) {
	if err := ValidateGoalGateConfig(req.Config); err != nil {
		return failGoalGate(req, "config", "goal_gate_config_invalid", err)
	}
	if !req.Config.Enabled {
		return GoalGateResult{Enabled: false, Triggered: false, SkippedReason: GoalGateSkipDisabled}, nil
	}
	effectiveSpec, err := BuildEffectiveGoalGateSpec(req.Spec, req.Config)
	if err != nil {
		return failGoalGate(req, "spec", "goal_gate_spec_invalid", err)
	}
	if strings.TrimSpace(req.Event.Hook) != steward.GoalBeforeCompleteHook {
		return failGoalGate(req, "event", "goal_gate_event_invalid", fmt.Errorf("goal gate event hook must be %q", steward.GoalBeforeCompleteHook))
	}
	if err := applyGoalGateDefaults(&req.Event, req.Config); err != nil {
		return failGoalGate(req, "event", "goal_gate_event_invalid", err)
	}
	now := time.Now().UTC()
	if req.NowUnix > 0 {
		now = time.Unix(req.NowUnix, 0).UTC()
	}
	var result steward.GoalGateSidecarResult
	if strings.TrimSpace(req.Config.Runner.Workdir) != "" {
		result, err = steward.ExecuteGoalGateSidecarInDir(
			ctx,
			effectiveSpec,
			req.Event,
			req.Audit,
			now,
			req.Config.Runner.Command,
			req.Config.Runner.Workdir,
			req.Config.Runner.Args...,
		)
	} else {
		result, err = steward.ExecuteGoalGateSidecar(
			ctx,
			effectiveSpec,
			req.Event,
			req.Audit,
			now,
			req.Config.Runner.Command,
			req.Config.Runner.Args...,
		)
	}
	if err != nil {
		stage, code := classifyGoalGateSidecarError(err)
		return failGoalGate(req, stage, code, err)
	}
	appendRecord := result.AppendRecord
	return GoalGateResult{
		Enabled:      true,
		Triggered:    true,
		Sidecar:      &result,
		AppendRecord: &appendRecord,
	}, nil
}

func classifyGoalGateSidecarError(err error) (stage string, code string) {
	message := strings.TrimSpace(err.Error())
	switch {
	case strings.HasPrefix(message, "proposal:") ||
		strings.HasPrefix(message, "decode agent proposal:") ||
		strings.Contains(message, "proposal hook"):
		return "proposal", "goal_gate_proposal_invalid"
	case strings.HasPrefix(message, "event:") ||
		strings.HasPrefix(message, "goal review requires") ||
		strings.HasPrefix(message, "goal review inputs") ||
		strings.HasPrefix(message, "decode goal_state:"):
		return "event", "goal_gate_event_invalid"
	default:
		return "runner", "goal_gate_runner_failed"
	}
}

func failGoalGate(req GoalGateRequest, stage string, code string, err error) (GoalGateResult, error) {
	result := GoalGateResult{
		Enabled:   req.Config.Enabled,
		Triggered: req.Config.Enabled,
		Failure: &GoalGateFailure{
			Stage:          stage,
			Code:           code,
			Message:        strings.TrimSpace(err.Error()),
			RunnerCommand:  strings.TrimSpace(req.Config.Runner.Command),
			RunnerWorkdir:  strings.TrimSpace(req.Config.Runner.Workdir),
			RunnerArgsHash: hashGoalGateRunnerArgs(req.Config.Runner.Args),
		},
	}
	if record, buildErr := buildGoalGateFailureAppendRecord(req, result.Failure); buildErr == nil {
		result.AppendRecord = &record
	}
	return result, &GoalGateExecutionError{Result: result, Err: err}
}

func buildGoalGateFailureAppendRecord(req GoalGateRequest, failure *GoalGateFailure) (ledger.AppendRecord, error) {
	if failure == nil {
		return ledger.AppendRecord{}, fmt.Errorf("goal gate failure is required")
	}
	metadata := map[string]string{
		"stage": failure.Stage,
	}
	if failure.RunnerCommand != "" {
		metadata["runner_command"] = failure.RunnerCommand
	}
	if failure.RunnerWorkdir != "" {
		metadata["runner_workdir"] = failure.RunnerWorkdir
	}
	if failure.RunnerArgsHash != "" {
		metadata["runner_args_hash"] = failure.RunnerArgsHash
	}
	if hook := strings.TrimSpace(req.Event.Hook); hook != "" {
		metadata["hook"] = hook
	}
	outcome := steward.GoalGateOutcome{
		AllowComplete:        false,
		ContinueWork:         false,
		BlockedReason:        failure.Message,
		LedgerEventType:      "error",
		LedgerEventAction:    "goal.review.failed",
		LedgerEventErrorCode: failure.Code,
		LedgerEventMetadata:  metadata,
	}
	return steward.BuildGoalGateAppendRecord(req.Audit, outcome)
}

func hashGoalGateRunnerArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(args, "\x00")))
	return fmt.Sprintf("sha256:%x", sum)
}

func BuildEffectiveGoalGateSpec(spec steward.Spec, cfg GoalGateConfig) (steward.Spec, error) {
	if err := ValidateGoalGateConfig(cfg); err != nil {
		return steward.Spec{}, err
	}
	if !cfg.Enabled {
		return spec, nil
	}
	if err := steward.Validate(spec); err != nil {
		return steward.Spec{}, fmt.Errorf("goal gate spec: %w", err)
	}
	if !containsValue(spec.Hooks, cfg.Hook) {
		return steward.Spec{}, fmt.Errorf("goal gate spec does not enable hook %q", cfg.Hook)
	}
	for _, input := range cfg.AllowedInputs {
		if !containsValue(spec.Inputs, input) {
			return steward.Spec{}, fmt.Errorf("goal gate config input %q is not declared by steward %q", input, spec.Name)
		}
	}
	for _, action := range cfg.AllowedActions {
		if !containsValue(spec.AllowedActions, action) {
			return steward.Spec{}, fmt.Errorf("goal gate config action %q is not allowed by steward %q", action, spec.Name)
		}
	}
	effective := spec
	effective.Hooks = []string{cfg.Hook}
	effective.Inputs = append([]string(nil), cfg.AllowedInputs...)
	effective.AllowedActions = append([]string(nil), cfg.AllowedActions...)
	return effective, nil
}

func applyGoalGateDefaults(event *steward.Event, cfg GoalGateConfig) error {
	var inputs map[string]json.RawMessage
	if err := json.Unmarshal(event.Inputs, &inputs); err != nil {
		return fmt.Errorf("decode goal gate event inputs: %w", err)
	}
	allowed := make(map[string]bool, len(cfg.AllowedInputs))
	for _, input := range cfg.AllowedInputs {
		allowed[input] = true
	}
	for key := range inputs {
		if !allowed[key] {
			return fmt.Errorf("goal gate event input %q is not enabled by runtime config", key)
		}
	}
	goalStateRaw, ok := inputs["goal_state"]
	if !ok {
		return fmt.Errorf("goal gate event inputs.goal_state is required")
	}
	var state steward.GoalState
	if err := json.Unmarshal(goalStateRaw, &state); err != nil {
		return fmt.Errorf("decode goal gate event goal_state: %w", err)
	}
	if state.MaxContinueAttempts == 0 && cfg.MaxContinueAttempts > 0 {
		state.MaxContinueAttempts = cfg.MaxContinueAttempts
	}
	if state.CooldownSeconds == 0 && cfg.CooldownSeconds > 0 {
		state.CooldownSeconds = cfg.CooldownSeconds
	}
	encodedState, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode goal gate event goal_state: %w", err)
	}
	inputs["goal_state"] = encodedState
	encodedInputs, err := json.Marshal(inputs)
	if err != nil {
		return fmt.Errorf("encode goal gate event inputs: %w", err)
	}
	event.Inputs = encodedInputs
	return nil
}

func validateGoalGateInputs(values []string) error {
	supported := map[string]bool{
		"goal_state":           true,
		"work_summary":         true,
		"test_results":         true,
		"blockers":             true,
		"recent_trace":         true,
		"changed_files":        true,
		"verification_summary": true,
		"user_goal":            true,
	}
	return validateSet("goal_gate_input", values, supported)
}

func validateGoalGateActions(values []string) error {
	supported := map[string]bool{
		"goal.approve_complete":  true,
		"goal.reject_complete":   true,
		"goal.request_continue":  true,
		"diagnosis.note.create":  true,
		"ledger.artifact.create": true,
		"context.inject":         true,
	}
	return validateSet("goal_gate_action", values, supported)
}

func containsValue(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}
