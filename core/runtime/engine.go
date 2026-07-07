package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	"github.com/nuctori/gateway-harness/core/policy"
	"github.com/nuctori/gateway-harness/core/trace"
)

type Evaluation struct {
	Decision plan.Decision `json:"decision"`
	Trace    trace.Trace   `json:"trace"`
}

type Engine struct {
	Policy policy.Policy
}

func NewEngine(p policy.Policy) Engine {
	return Engine{Policy: p}
}

func (e Engine) Evaluate(ev event.Event) (Evaluation, error) {
	tr := trace.Trace{
		TraceID:   ev.TraceID,
		RequestID: ev.RequestID,
		Events: []trace.EventRecord{{
			Type:  string(ev.Type),
			Model: ev.Model,
		}},
	}
	decision := plan.Decision{
		TraceID:       ev.TraceID,
		EventType:     string(ev.Type),
		MatchedPolicy: e.Policy.Name,
	}
	if !scopeMatches(e.Policy.Scope, ev) {
		decision.SkippedReason = "policy scope did not match event"
		tr.Decisions = append(tr.Decisions, trace.DecisionRecord{
			Policy: e.Policy.Name,
			Reason: decision.SkippedReason,
		})
		return Evaluation{Decision: decision, Trace: tr}, nil
	}
	rules := e.Policy.Hooks[string(ev.Type)]
	if len(rules) == 0 {
		decision.SkippedReason = "no hook matched event type"
		tr.Decisions = append(tr.Decisions, trace.DecisionRecord{
			Policy: e.Policy.Name,
			Reason: decision.SkippedReason,
		})
		return Evaluation{Decision: decision, Trace: tr}, nil
	}
	for _, rule := range rules {
		conditions, ok := conditionMatches(rule.If, ev)
		if !ok {
			tr.Decisions = append(tr.Decisions, trace.DecisionRecord{
				Policy:     e.Policy.Name,
				Reason:     "rule condition did not match",
				Conditions: conditions,
			})
			continue
		}
		if rule.Program != nil {
			return e.evaluateContextProgram(ev, rule, tr)
		}
		action, outcome, err := resolveAction(rule, ev)
		if err != nil {
			return Evaluation{}, err
		}
		decision.Actions = append(decision.Actions, action)
		tr.Decisions = append(tr.Decisions, trace.DecisionRecord{
			Policy:     e.Policy.Name,
			Action:     action.Type,
			FromModel:  action.FromModel,
			ToModel:    action.ToModel,
			Reason:     action.Reason,
			Conditions: conditions,
		})
		tr.Outcomes = append(tr.Outcomes, outcome)
		return Evaluation{Decision: decision, Trace: tr}, nil
	}
	decision.SkippedReason = "no rule condition matched"
	return Evaluation{Decision: decision, Trace: tr}, nil
}

func (e Engine) evaluateContextProgram(ev event.Event, rule policy.Rule, tr trace.Trace) (Evaluation, error) {
	decision := plan.Decision{
		TraceID:       ev.TraceID,
		EventType:     string(ev.Type),
		MatchedPolicy: e.Policy.Name,
	}
	limits := rule.Program.Budget.WithDefaults()
	patch := plan.ContextPatch{
		Summary: plan.PatchSummary{ContentMode: "redacted"},
	}
	var conditionLog []trace.ConditionRecord
	var added int64
	for _, step := range rule.Program.Steps {
		conditions, ok := conditionMatches(step.When, ev)
		conditionLog = append(conditionLog, conditions...)
		if !ok {
			continue
		}
		for _, action := range step.Do {
			op, tokenDelta, err := contextActionToPatch(action)
			if err != nil {
				return Evaluation{}, err
			}
			added += tokenDelta
			if limits.MaxPatchOps > 0 && len(patch.Operations)+1 > limits.MaxPatchOps {
				return Evaluation{}, fmt.Errorf("context patch op limit exceeded: max %d", limits.MaxPatchOps)
			}
			if limits.MaxAddedTokens > 0 && added > limits.MaxAddedTokens {
				return Evaluation{}, fmt.Errorf("context added token budget exceeded: max %d", limits.MaxAddedTokens)
			}
			patch.Operations = append(patch.Operations, op)
		}
	}
	patch.Summary.Ops = len(patch.Operations)
	patch.Summary.AddedEstimatedTokens = added
	decision.ContextPatch = &patch
	tr.Decisions = append(tr.Decisions, trace.DecisionRecord{
		Policy:         e.Policy.Name,
		Action:         "context.patch",
		MatchedProgram: e.Policy.Name,
		Conditions:     conditionLog,
		PatchSummary: &plan.PatchSummary{
			Ops:                  patch.Summary.Ops,
			AddedEstimatedTokens: patch.Summary.AddedEstimatedTokens,
			ContentMode:          patch.Summary.ContentMode,
		},
	})
	tr.Outcomes = append(tr.Outcomes, trace.Outcome{Type: "context.patch_applied"})
	return Evaluation{Decision: decision, Trace: tr}, nil
}

func scopeMatches(scope policy.Scope, ev event.Event) bool {
	if len(scope.Models) > 0 && !matchesAny(scope.Models, ev.Model) {
		return false
	}
	for _, required := range scope.Tags {
		if !ev.HasTag(required) {
			return false
		}
	}
	return true
}

func conditionMatches(cond policy.Condition, ev event.Event) ([]trace.ConditionRecord, bool) {
	var records []trace.ConditionRecord
	ok := true
	if len(cond.StatusIn) > 0 {
		status := 0
		if ev.Error != nil {
			status = ev.Error.Status
		}
		matched := false
		for _, allowed := range cond.StatusIn {
			if status == allowed {
				matched = true
				break
			}
		}
		records = append(records, trace.ConditionRecord{Expr: fmt.Sprintf("status_in:%v", cond.StatusIn), Result: matched})
		ok = ok && matched
	}
	if len(cond.MessageContains) > 0 {
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Message
		}
		matched := false
		for _, needle := range cond.MessageContains {
			if strings.Contains(strings.ToLower(msg), strings.ToLower(needle)) {
				matched = true
				break
			}
		}
		records = append(records, trace.ConditionRecord{Expr: fmt.Sprintf("message_contains:%v", cond.MessageContains), Result: matched})
		ok = ok && matched
	}
	if cond.ModelMatches != "" {
		matched := matchModel(cond.ModelMatches, ev.Model)
		records = append(records, trace.ConditionRecord{Expr: "model_matches:" + cond.ModelMatches, Result: matched})
		ok = ok && matched
	}
	if cond.EstimatedTokensGT > 0 {
		got := int64(0)
		if ev.Context != nil {
			got = ev.Context.EstimatedTokens
		}
		matched := got > cond.EstimatedTokensGT
		records = append(records, trace.ConditionRecord{Expr: fmt.Sprintf("estimated_tokens_gt:%d", cond.EstimatedTokensGT), Result: matched})
		ok = ok && matched
	}
	return records, ok
}

func resolveAction(rule policy.Rule, ev event.Event) (plan.Action, trace.Outcome, error) {
	switch rule.Action {
	case "prompt.append_system":
		return plan.Action{
			Type:   rule.Action,
			Text:   rule.Text,
			Reason: "matched prompt injection policy",
		}, trace.Outcome{Type: "decision.applied"}, nil
	case "fallback.sequence":
		next, ok := nextModel(rule.Models, ev.Model)
		if !ok {
			return plan.Action{
				Type:      "retry.with_model",
				FromModel: ev.Model,
				Reason:    "fallback.sequence exhausted",
			}, trace.Outcome{Type: "retry.exhausted", FromModel: ev.Model, Reason: "no later model in fallback sequence"}, nil
		}
		return plan.Action{
			Type:      "retry.with_model",
			FromModel: ev.Model,
			ToModel:   next,
			Reason:    "matched fallback.sequence",
		}, trace.Outcome{Type: "model.switched", FromModel: ev.Model, ToModel: next}, nil
	case "retry.with_model":
		if len(rule.Models) == 0 {
			return plan.Action{}, trace.Outcome{}, fmt.Errorf("retry.with_model needs at least one target model")
		}
		return plan.Action{
			Type:      rule.Action,
			FromModel: ev.Model,
			ToModel:   rule.Models[0],
			Reason:    "matched retry.with_model",
		}, trace.Outcome{Type: "model.switched", FromModel: ev.Model, ToModel: rule.Models[0]}, nil
	default:
		return plan.Action{}, trace.Outcome{}, fmt.Errorf("unsupported action %q", rule.Action)
	}
}

func contextActionToPatch(action policy.Action) (plan.PatchOperation, int64, error) {
	switch action.Action {
	case "context.inject":
		contentHash := sha256.Sum256([]byte(action.Text))
		return plan.PatchOperation{
			Op:          "append",
			Target:      "messages",
			Role:        action.Role,
			Position:    action.Position,
			Content:     action.Text,
			ContentHash: hex.EncodeToString(contentHash[:]),
			Reason:      action.Reason,
		}, estimateTokens(action.Text), nil
	case "context.truncate":
		return plan.PatchOperation{
			Op:                 "truncate",
			Target:             "messages",
			Strategy:           action.Strategy,
			KeepLastMessages:   action.KeepLastMessages,
			PreserveRoles:      action.PreserveRoles,
			MaxEstimatedTokens: action.MaxEstimatedTokens,
			Reason:             action.Reason,
		}, 0, nil
	case "context.remove", "context.reorder", "context.pin", "context.require", "context.reject":
		return plan.PatchOperation{
			Op:     strings.TrimPrefix(action.Action, "context."),
			Target: action.Target,
			Reason: action.Reason,
		}, 0, nil
	default:
		return plan.PatchOperation{}, 0, fmt.Errorf("unsupported context action %q", action.Action)
	}
}

func nextModel(models []string, current string) (string, bool) {
	for i, model := range models {
		if model == current {
			if i+1 < len(models) {
				return models[i+1], true
			}
			return "", false
		}
	}
	if len(models) > 0 {
		return models[0], true
	}
	return "", false
}

func matchesAny(patterns []string, model string) bool {
	for _, pattern := range patterns {
		if matchModel(pattern, model) {
			return true
		}
	}
	return false
}

func matchModel(pattern string, model string) bool {
	if pattern == "*" || pattern == model {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(model, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

func estimateTokens(text string) int64 {
	if text == "" {
		return 0
	}
	return int64((len([]rune(text)) + 3) / 4)
}
