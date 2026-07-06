package hooks

type LocaleText struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Example string `json:"example,omitempty"`
}

type CatalogEntry struct {
	ID                     string                `json:"id"`
	Family                 string                `json:"family"`
	Kind                   string                `json:"kind"`
	SupportsRequestPayload bool                  `json:"supports_request_payload"`
	GoalGateSelectable     bool                  `json:"goal_gate_selectable"`
	HostSurfaces           []string              `json:"host_surfaces"`
	Locales                map[string]LocaleText `json:"locales"`
}

var catalog = []CatalogEntry{
	{
		ID:                     "context.continuity_drop.detected",
		Family:                 "context",
		Kind:                   "signal",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "上下文连续性下降",
				Summary: "当宿主检测到会话上下文明显断裂时触发，适合显式注入短摘要或连续性提醒。",
				Example: "例如：压缩后发现上下文断崖下降，再补一条简短的项目约束摘要。",
			},
			"en-US": {
				Title:   "Context Continuity Drop",
				Summary: "Fires when the host detects a meaningful continuity loss, so policy can inject a short reminder or summary explicitly.",
				Example: "Example: after compaction, inject a short project-constraint summary when continuity drops sharply.",
			},
		},
	},
	{
		ID:                     "request.before_model_mapping",
		Family:                 "request",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "请求模型映射前",
				Summary: "在宿主把逻辑模型名映射到具体上游模型前运行，适合按逻辑模型或标签做显式上下文编程。",
				Example: "例如：在 failover 或别名展开前，先按模型标签注入固定系统提示词。",
			},
			"en-US": {
				Title:   "Request Before Model Mapping",
				Summary: "Runs before the host maps a logical request model to an upstream model, which is useful for model-family-aware context programming.",
				Example: "Example: inject a stable system capsule before failover aliases or channel routing expand the model name.",
			},
		},
	},
	{
		ID:                     "request.before_upstream",
		Family:                 "request",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "请求上游前",
				Summary: "在最终请求发往上游前运行，适合做最后一层显式上下文注入或审计补丁。",
				Example: "例如：在真正转发给模型前追加项目边界、输出格式或安全提醒。",
			},
			"en-US": {
				Title:   "Request Before Upstream",
				Summary: "Runs immediately before the final request leaves for the upstream provider, making it the last explicit mutation point.",
				Example: "Example: append project constraints, format instructions, or safety reminders just before forwarding.",
			},
		},
	},
	{
		ID:                     "chat.before_model_mapping",
		Family:                 "chat",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "Chat 模型映射前",
				Summary: "面向 Chat Completions 请求，在模型映射前做显式上下文编程。",
				Example: "例如：对某类 chat 模型统一补一条中文输出约束。",
			},
			"en-US": {
				Title:   "Chat Before Model Mapping",
				Summary: "Chat Completions variant of the pre-mapping hook, for explicit mutation before the host resolves the upstream model.",
				Example: "Example: add a stable language or formatting capsule for a chat-model family before routing.",
			},
		},
	},
	{
		ID:                     "chat.before_upstream",
		Family:                 "chat",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "Chat 上游前",
				Summary: "面向 Chat Completions 请求，在最终转发前做最后一层显式补丁。",
				Example: "例如：在发给上游前追加“保持工具调用协议不变”的提醒。",
			},
			"en-US": {
				Title:   "Chat Before Upstream",
				Summary: "Chat Completions variant of the final pre-upstream mutation point.",
				Example: "Example: add a last-mile reminder to preserve tool-call protocol before sending upstream.",
			},
		},
	},
	{
		ID:                     "responses.before_model_mapping",
		Family:                 "responses",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "Responses 模型映射前",
				Summary: "面向 Responses 请求，在宿主解析具体上游模型前运行，适合基于逻辑模型名注入约束。",
				Example: "例如：在 stateful Responses 路由前，为 coding 模型补充仓库约束。",
			},
			"en-US": {
				Title:   "Responses Before Model Mapping",
				Summary: "Responses endpoint variant of the pre-mapping hook for model-aware context programming.",
				Example: "Example: inject repository constraints for a coding model family before stateful Responses routing resolves upstream.",
			},
		},
	},
	{
		ID:                     "responses.before_upstream",
		Family:                 "responses",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "Responses 上游前",
				Summary: "面向 Responses 请求，在最终上游调用前运行，适合最后一跳的显式上下文注入。",
				Example: "例如：在请求真正出站前，补一条保持项目目标不漂移的提醒。",
			},
			"en-US": {
				Title:   "Responses Before Upstream",
				Summary: "Responses endpoint variant of the final pre-upstream mutation point.",
				Example: "Example: add a last-mile reminder to preserve the project goal before the request leaves the gateway.",
			},
		},
	},
	{
		ID:                     "responses.compact.before_model_mapping",
		Family:                 "responses.compact",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "压缩请求模型映射前",
				Summary: "专门面向 compact/压缩请求，在模型映射前运行，适合压缩场景的显式上下文归一化。",
				Example: "例如：在 compact 请求进上游前，统一注入“保留用户目标和未完成事项”的短摘要。",
			},
			"en-US": {
				Title:   "Compact Before Model Mapping",
				Summary: "Compact-request variant of the pre-mapping hook, for explicit continuity handling before model routing.",
				Example: "Example: inject a short capsule that preserves user intent and unfinished work before compact routing resolves upstream.",
			},
		},
	},
	{
		ID:                     "responses.compact.before_upstream",
		Family:                 "responses.compact",
		Kind:                   "mutation",
		SupportsRequestPayload: true,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"policy", "steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "压缩请求上游前",
				Summary: "专门面向 compact/压缩请求，在最终上游调用前运行，适合做最后一层连续性补丁。",
				Example: "例如：每次压缩事件发生时，显式注入由账本摘要提炼出的短提示词。",
			},
			"en-US": {
				Title:   "Compact Before Upstream",
				Summary: "Compact-request variant of the final pre-upstream hook, useful for the last explicit continuity patch before compaction leaves the gateway.",
				Example: "Example: inject a short prompt distilled from ledger summaries whenever a compact event is emitted.",
			},
		},
	},
	{
		ID:                     "goal.before_complete",
		Family:                 "goal",
		Kind:                   "decision",
		SupportsRequestPayload: false,
		GoalGateSelectable:     true,
		HostSurfaces:           []string{"steward", "goal_gate_host"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "完成前审查",
				Summary: "在执行器准备把目标标记为完成前触发，适合 AI in the loop 审查工作是否真的闭环。",
				Example: "例如：测试刚跑完时先让外部 reviewer 判断是否还缺部署、验收或回归验证。",
			},
			"en-US": {
				Title:   "Before Complete",
				Summary: "Fires when an executor is about to mark a goal complete, so an external reviewer can decide whether completion is justified.",
				Example: "Example: after tests pass, ask a reviewer whether deployment, acceptance, or regression checks are still missing.",
			},
		},
	},
	{
		ID:                     "goal.before_resume",
		Family:                 "goal",
		Kind:                   "decision",
		SupportsRequestPayload: false,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "恢复前审查",
				Summary: "在被拒绝、挂起或压缩后的目标准备恢复执行前触发，适合先归一化上下文，再决定下一步。",
				Example: "例如：继续上一轮编码前，先由 AI 检查 blockers、目标状态和最近 trace，再生成短继续提示。",
			},
			"en-US": {
				Title:   "Before Resume",
				Summary: "Fires before a rejected, paused, or compacted goal resumes, which is useful for continuity normalization before work continues.",
				Example: "Example: before resuming a coding task, let AI inspect blockers, goal state, and recent trace, then produce a short continuation prompt.",
			},
		},
	},
	{
		ID:                     "goal.after_complete",
		Family:                 "goal",
		Kind:                   "observation",
		SupportsRequestPayload: false,
		GoalGateSelectable:     false,
		HostSurfaces:           []string{"steward"},
		Locales: map[string]LocaleText{
			"zh-CN": {
				Title:   "完成后观察",
				Summary: "在目标已经完成后触发，适合补审计摘要、项目标签或后处理 artifact，而不是拦截完成动作本身。",
				Example: "例如：goal 真正完成后，再生成一条脱敏交付摘要并挂到 ledger。",
			},
			"en-US": {
				Title:   "After Complete",
				Summary: "Fires after a goal is already complete, which is useful for post-completion audit, tagging, or artifact generation rather than blocking completion itself.",
				Example: "Example: after the goal completes, generate a redacted delivery summary artifact and attach it to the ledger.",
			},
		},
	},
}

func Catalog() []CatalogEntry {
	out := make([]CatalogEntry, 0, len(catalog))
	for _, entry := range catalog {
		out = append(out, cloneEntry(entry))
	}
	return out
}

func SupportedMap() map[string]bool {
	out := make(map[string]bool, len(catalog))
	for _, entry := range catalog {
		out[entry.ID] = true
	}
	return out
}

func GoalGateSelectable() []CatalogEntry {
	var out []CatalogEntry
	for _, entry := range catalog {
		if entry.GoalGateSelectable {
			out = append(out, cloneEntry(entry))
		}
	}
	return out
}

func Lookup(id string) (CatalogEntry, bool) {
	for _, entry := range catalog {
		if entry.ID == id {
			return cloneEntry(entry), true
		}
	}
	return CatalogEntry{}, false
}

func cloneEntry(entry CatalogEntry) CatalogEntry {
	out := entry
	out.HostSurfaces = append([]string(nil), entry.HostSurfaces...)
	out.Locales = make(map[string]LocaleText, len(entry.Locales))
	for locale, text := range entry.Locales {
		out.Locales[locale] = text
	}
	return out
}
