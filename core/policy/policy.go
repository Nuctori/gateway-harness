package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Name  string            `json:"name" yaml:"name"`
	Scope Scope             `json:"scope" yaml:"scope"`
	Hooks map[string][]Rule `json:"hooks" yaml:"hooks"`
}

type Scope struct {
	Models []string `json:"models,omitempty" yaml:"models,omitempty"`
	Tags   []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type Rule struct {
	If      Condition       `json:"if,omitempty" yaml:"if,omitempty"`
	Action  string          `json:"action,omitempty" yaml:"action,omitempty"`
	Text    string          `json:"text,omitempty" yaml:"text,omitempty"`
	Models  []string        `json:"models,omitempty" yaml:"models,omitempty"`
	Program *ContextProgram `json:"program,omitempty" yaml:"program,omitempty"`
}

type ContextProgram struct {
	Budget ProgramBudget `json:"budget,omitempty" yaml:"budget,omitempty"`
	Steps  []ProgramStep `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type ProgramBudget struct {
	MaxEvalMS          int   `json:"max_eval_ms,omitempty" yaml:"max_eval_ms,omitempty"`
	MaxPatchOps        int   `json:"max_patch_ops,omitempty" yaml:"max_patch_ops,omitempty"`
	MaxAddedTokens     int64 `json:"max_added_tokens,omitempty" yaml:"max_added_tokens,omitempty"`
	MaxTraceMetadataKB int   `json:"max_trace_metadata_kb,omitempty" yaml:"max_trace_metadata_kb,omitempty"`
}

type ProgramStep struct {
	When    Condition `json:"when,omitempty" yaml:"when,omitempty"`
	Do      []Action  `json:"do,omitempty" yaml:"do,omitempty"`
}

type Action struct {
	Action             string   `json:"action,omitempty" yaml:"action,omitempty"`
	Role               string   `json:"role,omitempty" yaml:"role,omitempty"`
	Position           string   `json:"position,omitempty" yaml:"position,omitempty"`
	Text               string   `json:"text,omitempty" yaml:"text,omitempty"`
	Target             string   `json:"target,omitempty" yaml:"target,omitempty"`
	Strategy           string   `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	KeepLastMessages   int      `json:"keep_last_messages,omitempty" yaml:"keep_last_messages,omitempty"`
	PreserveRoles      []string `json:"preserve_roles,omitempty" yaml:"preserve_roles,omitempty"`
	MaxEstimatedTokens int64    `json:"max_estimated_tokens,omitempty" yaml:"max_estimated_tokens,omitempty"`
	Reason             string   `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type Condition struct {
	StatusIn          []int    `json:"status_in,omitempty" yaml:"status_in,omitempty"`
	MessageContains   []string `json:"message_contains,omitempty" yaml:"message_contains,omitempty"`
	ModelMatches      string   `json:"model_matches,omitempty" yaml:"model_matches,omitempty"`
	EstimatedTokensGT int64    `json:"estimated_tokens_gt,omitempty" yaml:"estimated_tokens_gt,omitempty"`
}

func (b ProgramBudget) WithDefaults() ProgramBudget {
	if b.MaxEvalMS == 0 {
		b.MaxEvalMS = 20
	}
	if b.MaxPatchOps == 0 {
		b.MaxPatchOps = 16
	}
	if b.MaxAddedTokens == 0 {
		b.MaxAddedTokens = 1200
	}
	if b.MaxTraceMetadataKB == 0 {
		b.MaxTraceMetadataKB = 32
	}
	return b
}

func LoadFile(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	switch ext := filepath.Ext(path); ext {
	case ".json":
		err = json.Unmarshal(raw, &p)
	case ".yaml", ".yml", "":
		err = yaml.Unmarshal(raw, &p)
	default:
		err = yaml.Unmarshal(raw, &p)
	}
	if err != nil {
		return Policy{}, err
	}
	return p, p.Validate()
}

func (p Policy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if len(p.Hooks) == 0 {
		return fmt.Errorf("policy hooks are required")
	}
	for hook, rules := range p.Hooks {
		if hook == "" {
			return fmt.Errorf("hook name is required")
		}
		if len(rules) == 0 {
			return fmt.Errorf("hook %q has no rules", hook)
		}
		for i, rule := range rules {
			if rule.Action == "" && rule.Program == nil {
				return fmt.Errorf("hook %q rule %d needs action or program", hook, i)
			}
			if rule.Action == "fallback.sequence" && len(rule.Models) == 0 {
				return fmt.Errorf("hook %q rule %d fallback.sequence needs models", hook, i)
			}
				if rule.Program != nil {
					if rule.Program.Budget.MaxPatchOps < 0 || rule.Program.Budget.MaxAddedTokens < 0 {
						return fmt.Errorf("hook %q rule %d has invalid negative budget", hook, i)
					}
					for j, step := range rule.Program.Steps {
						actions := step.Do
						if len(actions) == 0 {
							return fmt.Errorf("hook %q rule %d program step %d has no actions", hook, i, j)
						}
					for k, action := range actions {
						if action.Action == "" {
							return fmt.Errorf("hook %q rule %d program step %d action %d missing action", hook, i, j, k)
						}
					}
				}
			}
		}
	}
	return nil
}
