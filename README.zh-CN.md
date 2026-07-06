# Gateway Harness

[English README](README.md)

Gateway Harness 是一个面向 LLM 网关的、宿主无关的上下文编程层。

它的核心目标不是替代 NewAPI、LiteLLM、OpenRouter 或任何具体网关，而是把“什么时候介入请求、如何改写上下文、如何审计这次改写”抽象成一套可版本化、可验证、可被 WebUI 生成的策略契约。

NewAPI 在本仓库里只是 adapter 示例，不是 Gateway Harness 概念本身的所有者。

## 为什么需要它

普通网关通常擅长做这些事：

- API key 管理
- 额度与计费
- 模型映射
- 上游渠道选择
- 请求转发
- 日志与审计

但真实使用 LLM 时，还经常需要更细的“上下文控制”：

- 某个模型永远需要一段系统提示词。
- 某类模型只在压缩上下文时注入提醒。
- 请求走到上游前，需要根据模型、标签、token 估算或请求形态做一次上下文补丁。
- 发生压缩、裁剪、降级、模型切换时，希望上下文策略仍然可解释、可审计。
- WebUI 里不想让用户写一坨不知道字段含义的 JSON，而是希望能由 schema 和能力声明生成表单。

Gateway Harness 解决的是这一层：把上下文行为从“写死在某个网关代码里”提升为“独立策略 + adapter 执行”的契约。

## 核心概念

- `Policy`：一份策略文件，描述有哪些程序化上下文规则。
- `Program`：一组面向模型、标签或场景的规则集合。
- `Hook`：策略在哪个阶段执行，例如 `request.before_upstream`。
- `Action`：策略要做什么，默认心智模型是 `context.inject` 这类显式注入。
- `Condition`：策略什么时候生效，例如匹配模型或 token 估算。
- `Explicit Guard`：如果 adapter 需要限制或保护某个改写行为，必须把它声明成显式 action 或 adapter guard。
- `Trace`：记录脱敏审计信息，例如命中的 program、hook、操作数和内容 hash。
- `Adapter`：宿主网关的胶水层，例如 NewAPI adapter。
- `Adapter Capability`：adapter 显式声明自己支持哪些 hook、action、请求形态和 guard。
- `Ledger`：按项目、会话、事件组织的审计账本，只记录 hash、引用和脱敏元数据，不保存原始 prompt/response。
- `Steward`：显式配置的 AI-in-the-loop 上下文管家契约，用来提出可验证的 context patch 和审计 artifact。

## 当前已支持什么

v0.2 是一个可发布、可 CI 验收的契约版本：

- Go policy structs。
- Policy validation。
- Policy dry-run，输出脱敏 patch plan。
- Adapter capability structs。
- Adapter capability validation。
- Conformance fixture validation。
- 本地 fake upstream replay。
- Policy apply + replay，用于验证 adapter 风格改写后的真实请求形态。
- Project/session ledger validation。
- Project/session ledger query，可按 project、session、tag、event type 做脱敏审计检索。
- Normalized Rule validation/compile，把 `Trigger + Scope + Operation + Audit` 编译成底层 policy。
- AI-in-the-loop steward spec validation。
- AI-in-the-loop steward external runner：显式调用开源 agent runner，并校验其 proposal。
- AI-in-the-loop steward proposal validation。
- AI-in-the-loop steward proposal dry-run。
- JSON Schema。
- CLI：`validate`、`explain`、`schema`、`validate-rule`、`explain-rule`、`compile-rule`、`compile-rule-stewards`、`rule-schema`、`dry-run-policy`、`validate-adapter`、`explain-adapter`、`adapter-schema`、`goal-gate-config-schema`、`validate-goal-gate-config`、`validate-conformance`、`explain-conformance`、`replay-conformance`、`replay-policy-conformance`、`conformance-schema`、`validate-ledger`、`explain-ledger`、`query-ledger`、`append-ledger-record`、`ledger-schema`、`ledger-record-schema`、`validate-steward`、`explain-steward`、`steward-schema`、`steward-event-schema`、`validate-steward-event`、`run-steward`、`validate-steward-proposal`、`explain-steward-proposal`、`steward-proposal-schema`、`dry-run-steward-proposal`、`execute-goal-gate`。

Goal Gate 宿主配置也有独立 CLI：`goal-gate-config-schema`、`validate-goal-gate-config`、`execute-goal-gate`。这样 WebUI 或宿主进程即使不直接嵌入 Go 包，也能用同一份显式 config、steward spec、event 和 audit 输入来验证或执行 Goal Gate。
- NewAPI adapter contract 文档。
- NewAPI 示例 policy。
- Release 产物：Linux amd64、Linux arm64、Linux armv7、Windows amd64。

默认推荐的归一化 rule 当前有两个 operation：

- `inject_capsule`：注入一段显式上下文。填写 `audit.ledger_ref` 时，会编译成带 Ledger provenance 的注入。
- `ask_steward`：声明一个显式 AI-in-the-loop steward 契约。它只编译出 steward spec，不会调用 AI，也不会改写请求。

`ask_steward` 默认只允许非破坏性、可审计输出，例如 `context.inject`、`ledger.artifact.create`、`diagnosis.note.create` 和 `session.tags.update`。它不支持 `context.truncate`、`policy.patch.propose` 或“人工审批”伪能力；如果未来要做策略补丁审批，应作为独立上层 GitOps/Console/PR 工作流，而不是网关默认能力。

AI-in-the-loop 的执行边界也保持很薄：Gateway Harness 不内置 agent、不内置 provider、不偷偷调用 AI。`run-steward` 只显式启动一个外部开源 agent runner，把已校验、已脱敏且声明 `"redacted": true` 的事件 JSON 传给 stdin，再校验 stdout 返回的 steward proposal。事件只能包含 steward spec 声明过的 input 名，并递归拒绝 `prompt`、`messages`、`content`、`input`、`output` 等原始内容键。推荐示例见 `examples/smolagents/`。

底层 policy action 是编译目标和高级兼容层，不是默认 GUI 心智模型：

- `context.inject`：注入一段上下文。
- `context.inject_ledger_summary`：本质仍是注入，只是额外携带 `ledger.summary` 来源、`ledger_ref` 和 `artifact_refs`。
- `context.truncate`：破坏性删除上下文，只适合高级 JSON 显式 opt-in，不属于默认归一化 rule。

当前内置 hook：

- `context.continuity_drop.detected`
- `request.before_model_mapping`
- `request.before_upstream`
- `chat.before_model_mapping`
- `chat.before_upstream`
- `responses.before_model_mapping`
- `responses.before_upstream`
- `responses.compact.before_model_mapping`
- `responses.compact.before_upstream`
- `goal.before_complete`
- `goal.before_resume`
- `goal.after_complete`

这些 hook 已经进入统一 `hook_catalog`，并附带 `zh-CN` / `en-US` 标题、说明和示例，方便 WebUI 直接做 hover、图鉴或字段说明。
需要注意的是：当前 Goal Gate 宿主配置页仍然只开放 `goal.before_complete` 作为可执行拦截点；`goal.before_resume` 和 `goal.after_complete`
已经进入底层契约和目录元数据，但还没有被这个宿主表单直接暴露成可运行开关。

## 快速开始

验证一份 policy：

```bash
gateway-harness validate examples/newapi/context-harness.policy.json
```

解释一份 policy：

```bash
gateway-harness explain examples/newapi/context-harness.policy.json
```

打印 JSON Schema：

```bash
gateway-harness schema
```

验证一份归一化 rule：

```bash
gateway-harness validate-rule fixtures/newapi/context-rule.continuity-drop.json
```

把归一化 rule 编译成底层 policy：

```bash
gateway-harness compile-rule fixtures/newapi/context-rule.continuity-drop.json
```

把归一化 AI-in-loop rule 编译成 steward spec：

```bash
gateway-harness compile-rule-stewards fixtures/newapi/context-rule.ask-steward.json
```

打印 rule schema：

```bash
gateway-harness rule-schema
```

对请求副本 dry-run 一份 policy：

```bash
gateway-harness dry-run-policy examples/newapi/context-harness.policy.json responses.compact.before_upstream fixtures/newapi/policy-dry-run.request.json
```

验证 adapter capability：

```bash
gateway-harness validate-adapter examples/newapi/adapter.capability.json
```

打印 Goal Gate 宿主配置 schema：

```bash
gateway-harness goal-gate-config-schema
```

打印 Goal Gate 宿主表单 schema 和默认表单模型：

```bash
gateway-harness goal-gate-result-schema
gateway-harness goal-gate-form-schema
gateway-harness goal-gate-form-model
gateway-harness goal-gate-host-bundle
```

这份 schema 不是只给校验器看的，也可以直接给 WebUI / Console 渲染配置表单。里面已经包含
`hook`、`allowed_inputs`、`allowed_actions` 的标题、说明、示例和 GUI 友好的枚举元数据，宿主
不需要再偷偷维护一份看不见的字段字典。

`goal-gate-form-model` 是再往上一层的薄契约：它不改变底层 schema，只额外给出字段分组、推荐控件、展示顺序
和默认透明提示，让宿主可以直接拼出一个“人能看懂”的配置页，而不是自己发明一套隐藏布局逻辑。`goal-gate-host-bundle`
现在还会额外打包 `hook_catalog` 和多语言元数据，方便页面把 hook 解释、可用性边界和示例直接展示出来。
它同样可以作为“聊天式配置助手”的底座，但前提仍然是：AI 只能生成显式 config 草案，最终启用和保存必须是宿主的显式动作。

`goal-gate-result-schema` 则是输出侧对应的正式契约。它把 approve、reject/continue、结构化 failure、ledger append
以及可选 `continuation_patches` 的结果形态固定下来，宿主和 WebUI 不需要再靠读 Go 结构体猜字段。

`goal-gate-host-bundle` 是进一步收敛后的“一次性接线包”：它会把 config schema、form schema、form model、result schema、
聊天式交互元数据以及 approve / reject / failure 示例结果一起打包成一个 JSON，方便宿主或页面初始化时一次拉全，不用再自己协调多次 CLI 调用。

如果你想按验收标准逐条核对当前实现，可直接看 [docs/goal-gate-acceptance.md](/C:/Users/Nuctori/Documents/Codex/2026-07-03/c-new-api-build/work/gateway-harness/docs/goal-gate-acceptance.md)。

验证 Goal Gate 宿主配置：

```bash
gateway-harness validate-goal-gate-config examples/newapi/goal-gate.config.json
```

`runner.workdir` 如果写成相对路径，会在执行前按 config 文件所在目录解析，避免 Goal Gate 的行为依赖调用方当前 shell 的工作目录。
如果 `execute-goal-gate` 失败，它仍然会先打印一份结构化 `GoalGateResult`；当 audit 输入足够时，
结果里还会带可直接追加到 ledger 的 `append_record`，对应一条显式 `error` 事件，然后进程再以非 0 退出。

验证 conformance fixture：

```bash
gateway-harness validate-conformance fixtures/newapi/responses-tool-chain.conformance.json
```

用本地 fake upstream 回放 conformance fixture：

```bash
gateway-harness replay-conformance fixtures/newapi/responses-tool-chain.conformance.json
```

先应用 fixture 里的 policy，再把改写后的请求副本回放到本地 fake upstream，并且只打印脱敏 trace：

```bash
gateway-harness replay-policy-conformance fixtures/newapi/responses-policy-apply.conformance.json responses.before_upstream
```

打印 conformance fixture schema：

```bash
gateway-harness conformance-schema
```

验证项目/会话审计 ledger：

```bash
gateway-harness validate-ledger fixtures/newapi/project-session.ledger.json
```

解释 ledger：

```bash
gateway-harness explain-ledger fixtures/newapi/project-session.ledger.json
```

按 project、session、tag 或 event type 查询持久化 ledger：

```bash
gateway-harness query-ledger fixtures/newapi/project-session.ledger.json -tag adapter:newapi -tag domain:coding -event-type compact
```

把一个显式、脱敏的事件 record 追加进项目/会话 ledger；目标 ledger、project 或 session 不存在时会创建：

```bash
gateway-harness append-ledger-record runtime/project-session.ledger.json fixtures/newapi/continuity-drop.append-record.json
```

打印 ledger schema：

```bash
gateway-harness ledger-schema
```

打印 ledger append-record schema：

```bash
gateway-harness ledger-record-schema
```

验证 AI steward spec：

```bash
gateway-harness validate-steward fixtures/newapi/compact-context.steward.json
```

解释 steward spec：

```bash
gateway-harness explain-steward fixtures/newapi/compact-context.steward.json
```

打印 steward schema：

```bash
gateway-harness steward-schema
```

用 steward spec 验证 AI 返回的 proposal：

```bash
gateway-harness validate-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json
```

解释 steward proposal：

```bash
gateway-harness explain-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json
```

打印 steward proposal schema：

```bash
gateway-harness steward-proposal-schema
```

对请求副本 dry-run 一个 steward proposal：

```bash
gateway-harness dry-run-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json fixtures/newapi/compact-context.request.json
```

验证一个 `goal.before_complete` steward event：

```bash
gateway-harness validate-steward-event fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json
```

验证 Goal Gate 的 approve / reject proposal：

```bash
gateway-harness validate-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.approve.json

gateway-harness validate-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

对 Goal Gate 的 reject proposal 做 dry-run：

```bash
gateway-harness dry-run-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json fixtures/goal-gate/goal.before_complete.request.json
```

把 Goal Gate proposal 解释成最终判定结果：

```bash
gateway-harness evaluate-goal-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

把判定结果推进成宿主可消费的下一份 `goal_state`：

```bash
gateway-harness apply-goal-review-result fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

直接用显式 config + steward spec + event + audit 输入执行完整的宿主侧 Goal Gate：

```bash
gateway-harness execute-goal-gate examples/newapi/goal-gate.config.json fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.audit.json
```

在没有真实 NewAPI 主机和 API key 的 CI 环境里跑 NewAPI 在线验收脚本的 mock 测试：

```bash
sh examples/newapi/online-acceptance.test.sh
```

这个测试会创建临时 SQLite 配置库和假的 `curl` / `docker` / `gateway-harness` 命令，覆盖默认无 token 路径、live smoke、compact smoke、端口检查、脱敏 trace 检查、failover 配置校验，以及失败响应体默认不打印。

Conformance fixture 验证的是 Gateway Harness 契约、adapter capability 和真实请求形态。`replay-conformance` 会把 fixture request 通过 HTTP POST 发到本地 fake upstream，不触网、不调用真实模型；它仍不替代 live upstream 端到端测试。

`replay-policy-conformance` 会更进一步：先把 policy 应用到请求副本，再把改写后的请求发到本地 fake upstream，只输出请求大小和脱敏 trace。它适合放进 CI，用来证明 adapter 风格的真实改写没有破坏协议形态，也没有把原始 prompt 或原始注入文本写进日志输出。

Ledger 验证的是“项目/会话/事件/摘要 artifact”这层审计边界。它不保存原始对话内容，只保存事件元数据、`content_hash` 和外部引用，方便后续接 sidecar、SQLite、对象存储或向量索引。Metadata 只适合放标签和 ID；`prompt`、`response`、`messages` 这类明显承载原文的 key 会被拒绝。
`query-ledger` 让这些持久化 session 可以按项目、会话、标签和事件类型检索，但输出仍只包含 session 元数据、事件计数、命中的 event id 和 artifact id，不做原文搜索。
`append-ledger-record` 是适配器和 sidecar 的最小持久化原语：它原子追加一个显式事件和可选的 hash artifact，可以创建 project/session 边界，但仍会拒绝 raw prompt/response 元数据，并在写回前校验完整 ledger。

Steward 验证的是“AI 可以参与上下文管理”的显式边界。它可以声明在 compact、failover 或诊断 hook 上唤起 AI，但必须使用显式 hook、脱敏输入、结构化输出、可校验 action 和带 hash 的 artifact。Gateway Harness core 不会因为存在 steward schema 就默认调用 AI；实际调用必须由 adapter 或 sidecar 明确实现。

Steward proposal 验证的是“AI 实际返回了什么”。Proposal 必须拿对应 spec 一起校验，因此 AI 不能使用未启用的 hook、不能输出未允许的 action、不能创建没有 hash 的 artifact，也不能静默应用 policy patch。

`dry-run-steward-proposal` 只打印非破坏性 proposal 输出的脱敏 patch plan。它不打印完整改写请求、不调用 AI、不写数据库、不请求上游，也不会便利执行 `context.truncate` 这种破坏性上下文编辑。

对 Goal Gate 而言，`goal.approve_complete`、`goal.reject_complete` 和 `goal.request_continue` 在 dry-run 里会以结构化 `goal_actions` 形式输出，而不会直接改写请求副本。真实的 complete 拦截、续跑次数上限、cooldown 和去重逻辑，仍然必须由显式的 adapter 或 sidecar 集成来承接。

对 adapter / sidecar 而言，`evaluate-goal-proposal` 和 `apply-goal-review-result` 就是建立在 steward 契约之上的薄执行层：前者决定这个 goal 是否允许 complete，后者把这个判定推进成显式的下一份 `goal_state`、继续/停止工作结论，以及可落账本的脱敏审计元数据。

如果宿主是进程内集成，也可以直接调用 `ReviewGoalCompletion`、`EvaluateGoalProposal`、`ApplyGoalReviewResult` 和 `BuildGoalGateAppendRecord`。这样 sidecar 可以把 Goal Gate 全链路保持为显式流程：调用配置好的 runner、校验结构化 proposal、推进下一份持久化 `goal_state`，最后再把一条脱敏的 `harness_action` 或 `error` 事件追加进 session ledger，而不是在宿主里再发明一套隐藏状态机。

`dry-run-policy` 把同样的透明性边界用于普通 policy。它只输出命中的 program、会执行的 action、被跳过的破坏性 action，以及脱敏 request patch 元数据，例如 target、insert index、role、reason、content hash 和内容长度。它不打印完整改写请求，也不打印原始注入文本。带 `estimated_tokens_gt` 的条件只有在调用方显式传入 `estimated_tokens` 参数时才会命中。

对压缩感知 adapter，推荐用 `context.inject_ledger_summary` 做显式 sentinel。adapter 或操作者必须提供要注入的摘要文本，并声明 `ledger_ref` 和可选的 `artifact_refs`；Gateway Harness 只应用这次显式 patch，并在 dry-run/trace 里记录脱敏来源、hash 和长度。

## 示例 policy

```json
{
  "version": "0.1",
  "programs": [
    {
      "name": "coding-model-harness",
      "models": ["gpt-5.4-mini", "kimi-for-coding"],
      "tags": ["domain:coding"],
      "steps": [
        {
          "hook": "responses.before_upstream",
          "when": {
            "model_matches": "*"
          },
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "reason": "preserve coding constraints",
              "text": "Preserve user intent, repository constraints, and prior architecture decisions."
            }
          ]
        }
      ]
    }
  ]
}
```

## 典型用例

### 1. 按模型注入系统提示词

给某些模型补一段稳定提示词，例如：

- 旗舰模型更强调架构正确性。
- flash 模型更强调简洁执行。
- coding 模型更强调保留仓库约束。

这类策略适合放在 `chat.before_upstream` 或 `responses.before_upstream`。

### 2. 按模型家族注入不同策略

可以把模型分成不同 program：

- `gpt-*`：面向 OpenAI 风格 Responses。
- `kimi-*`：面向长上下文代码分析。
- `deepseek-*`：面向低成本快速执行。
- `glm-*`：面向中文与结构化输出。

v0.1 用 `models` 和 `model_matches` 表达；后续 adapter 可以把 UI 做成模型下拉选择。

### 3. 压缩上下文时注入提醒

在 `responses.compact.before_upstream` 注入“压缩时必须保留什么”的提示，例如：

- 用户目标。
- 已做决策。
- 未解决问题。
- 仓库约束。
- 部署状态。
- 风险和回滚路径。

这适合避免上下文压缩后模型“忘记真正目标”。

### 4. 压缩 hook 触发外部总结器

更高级的 adapter 可以在压缩 hook 上调用外部 summarizer：

- 提取当前对话里的用户目标。
- 提取 AI 已执行的命令和结论。
- 生成短 prompt。
- 再通过 `context.inject` 注回请求。

这属于 v0.2+ 的 adapter 能力，不建议在 v0.1 里直接执行任意脚本。

### 5. 为降级链保留上下文一致性

模型降级本身通常属于网关路由或 failover 模块，不是 Gateway Harness 的核心 action。

但 Gateway Harness 可以配合降级链使用：

- 旗舰模型失败后切到同领域模型。
- 同领域模型都失败后切到跨领域兜底模型。
- 切换后仍注入“当前请求来自 failover，必须保持原用户意图”的上下文。

这样降级不只是换模型，还能让新模型理解为什么被切过来。

### 6. 按领域标签选择上下文策略

一个请求可以带标签，例如：

- `domain:coding`
- `domain:writing`
- `domain:search`
- `domain:vision`
- `tenant:team-a`
- `risk:high`

策略可以按标签组合不同上下文片段。v0.1 已有 `tags` 字段用于声明，后续 adapter 可把它接到路由、用户组或 UI。

### 7. WebUI 表单生成

JSON Schema 可以驱动 WebUI：

- hook 下拉选择。
- action 下拉选择。
- role 下拉选择。
- position 下拉选择。
- 字段 hover 说明。
- policy 校验。
- 示例模板生成。

这能让用户不用手写 JSON，也知道每个字段能填什么。

### 8. CI 中验证策略文件

在网关配置仓库里，可以把 policy 当成代码审查对象：

```bash
gateway-harness validate gateway-harness.policy.json
```

适合防止：

- 写错 hook。
- 写错 action。
- 忘记 models。
- 空文本注入。
- 使用了未声明字段。

### 9. 生成脱敏 trace

adapter 执行策略时应写入脱敏 trace：

- 命中的 program。
- 执行 hook。
- 操作数量。
- 新增 token 估算。
- 注入内容 hash。

不要把完整提示词直接写入日志，避免把用户数据或内部 prompt 泄漏出去。

### 10. 显式声明改写保护

Gateway Harness 不定义隐式 program-level budget。

如果某个 adapter 需要保护改写行为，例如限制某个 action 的影响范围，它应该把这种行为表达成显式 action 或 adapter guard。没有在 policy 或 adapter capability 中明确声明的行为，就不应该偷偷发生。

这条原则尤其重要：Gateway Harness 不应该因为自己的估算比上游模型小，就拒绝用户请求。

### 11. 安全裁剪上下文

`context.truncate` 可以保留最近若干条消息，并保留系统/开发者角色。

对 Responses 工具调用链要特别保守：如果请求里包含 `previous_response_id`、`function_call`、`function_call_output`、`item_reference` 或 `reasoning`，adapter 不应粗暴裁剪 input，否则可能破坏工具调用续写关系。

### 12. 为不同租户配置策略

多团队网关可以按租户挂不同策略：

- A 团队偏代码生成。
- B 团队偏中文写作。
- C 团队偏低成本摘要。
- 高风险租户强制审计 trace。

Gateway Harness 本身只定义策略结构，租户绑定由 adapter 或宿主网关负责。

### 13. 为不同 endpoint 配置策略

Chat Completions 和 Responses 的上下文结构不一样，因此 hook 应区分：

- `chat.before_upstream`
- `responses.before_upstream`
- `responses.compact.before_upstream`

adapter 负责把统一 action 映射到各 endpoint 的请求对象。

adapter 还应该发布 capability manifest，让策略作者和 WebUI 知道哪些 endpoint 形态真的可用。

### 14. 策略解释和审计

`gateway-harness explain` 可以给策略做摘要：

- 有几个 program。
- 有几个 step。
- 有几个 action。
- 用到了哪些 hook。

后续可以扩展成更完整的审计报告，例如“哪些模型会被哪些策略影响”。

### 15. 作为 adapter 能力契约

不同网关不一定支持同样的 hook 和 action。

正确做法是 adapter 声明自己支持的能力，WebUI 和 CLI 根据能力生成配置，而不是把所有字段都写死。

v0.2 起，这个能力矩阵可以用 `adapter.capability.json` 表达，并通过 `gateway-harness validate-adapter` 校验。

### 16. 灰度发布新的上下文策略

上下文策略和代码一样，也需要灰度：

- 先只对一个租户启用。
- 再对一个模型家族启用。
- 最后扩展到全局默认策略。

Gateway Harness 可以把策略拆成多个 program，adapter 决定哪些租户、模型或分组能看到这些 program。

### 17. A/B 测试提示词变体

同一个模型可以挂两套上下文策略：

- A 版本更保守。
- B 版本更主动。
- C 版本更短、更省 token。

Gateway Harness 负责声明变体和 trace；分流、采样和效果统计由宿主网关或观测系统负责。

### 18. 成本分层上下文

不同成本等级可以注入不同上下文：

- 高价模型拿完整约束。
- 中价模型拿压缩约束。
- 低价模型只拿关键边界。

这可以配合模型路由和 failover 使用，让降级后的模型仍知道任务底线，同时控制 prompt 成本。

### 19. 高风险请求加强约束

对 `risk:high` 或管理类请求，可以注入更强约束：

- 不泄露密钥。
- 不执行破坏性命令。
- 变更前说明影响面。
- 必须给出回滚路径。

Gateway Harness 不替代安全沙箱，但可以把安全意图稳定注入到模型上下文。

### 20. 隐私与日志脱敏策略

一些 adapter 可以在执行 trace 时只记录：

- content hash。
- action 类型。
- token 估算。
- program 名称。
- hook 名称。

这样既能审计策略是否生效，又避免把用户原文、内部 prompt 或业务数据写进日志。

### 21. RAG 检索前后注入约束

在带检索的网关里，可以把上下文策略放在检索前或检索后：

- 检索前：注入查询改写约束。
- 检索后：注入“只基于证据回答”的约束。
- 上游前：注入引用格式或不确定性声明。

v0.1 只定义 hook/action 契约；具体检索管线 hook 需要 adapter 扩展。

### 22. 会话标签与索引协作

后续系统可以把会话上下文按标签组织：

- 项目名。
- 仓库名。
- 用户目标。
- 任务阶段。
- 风险等级。

Gateway Harness 可读取这些标签来选择策略；标签的生成、索引和搜索属于宿主系统或外部记忆层。

### 23. 多模态请求的上下文提示

图像、音频、文件类请求也可能需要稳定约束：

- 图片分析时要求指出不确定性。
- 文件总结时保留章节结构。
- OCR 后保留原文引用。

Gateway Harness 可以声明这些策略；adapter 负责把它们映射到具体多模态 API 的可用字段。

### 24. 工具调用前的行为约束

模型调用工具前可以注入约束：

- 先解释为什么需要工具。
- 对 destructive tool 必须二次确认。
- 工具参数要最小化。
- 工具输出不能直接当最终答案。

这类能力需要 adapter 提供 tool-call 相关 hook。v0.1 的文档已明确 Responses 工具链不能被粗暴裁剪。

### 25. 工具调用后的结果整理

工具返回后可以注入“如何使用工具结果”的策略：

- 区分事实和推测。
- 保留错误信息。
- 不吞掉失败原因。
- 对多个工具结果做冲突检查。

这适合未来的 `tool.after_output` 或类似 hook，不应该在 v0.1 里靠字符串黑魔法模拟。

### 26. 面向小设备的轻量策略执行

像 ARMv7 小主机、边缘网关或家庭服务器，通常资源有限。

Gateway Harness 的策略执行应尽量：

- 不依赖重型运行时。
- 不强制数据库。
- 不强制网络调用。
- 可以只做本地 JSON 校验和轻量 context patch。

这也是为什么 v0.1 先提供 Go CLI、schema 和静态 release 产物。

### 27. 回滚策略

策略变更出问题时，应该能快速回滚：

- 回到上一份 policy。
- 禁用某个 program。
- 禁用某个 hook。
- 移除或关闭某个显式 guard。

Gateway Harness policy 适合进 Git 管理，这样上下文行为也能像代码一样 diff、review 和 rollback。

### 28. 策略模板市场

未来可以沉淀通用模板：

- coding 模板。
- writing 模板。
- summarization 模板。
- support 模板。
- safe-ops 模板。

模板应是普通 policy 片段，而不是绑死某个网关实现。

### 29. 能力发现与 UI 自动适配

理想的 adapter 应暴露能力矩阵：

- 支持哪些 hook。
- 支持哪些 action。
- 哪些字段可编辑。
- 哪些 action 有安全限制。
- 哪些 hook 只读、哪些 hook 可变更请求体。

WebUI 根据能力矩阵生成表单，避免用户看到一堆当前 adapter 根本不能执行的选项。

### 30. 解释“为什么这次请求被这样改写”

当用户问“为什么我的请求被注入了这段上下文”时，系统应该能回答：

- 命中了哪个 program。
- 触发了哪个 hook。
- 满足了哪个 condition。
- 执行了哪些 action。
- 注入内容的 hash 是什么。

这能把 Gateway Harness 从黑盒代理变成可解释的上下文层。

### 31. 上下文策略的单元测试

策略可以像代码一样测试：

- 给定模型名。
- 给定 hook。
- 给定 token 估算。
- 给定输入消息。
- 断言会执行哪些 action。

v0.1 CLI 先提供 validation；后续可以扩展 `gateway-harness test` 来执行 fixture。

### 32. 请求形态兼容性保护

不同 API 的请求形态有硬边界：

- Chat messages 可以安全裁剪消息尾部。
- Responses stateful continuation 可能依赖 item reference。
- Tool output 可能必须匹配 call id。
- Compact 请求可能有特殊入口。

Gateway Harness 的 adapter 必须理解这些边界，宁愿跳过某些 action，也不要破坏上游协议。

v0.2 起，contract conformance fixture 可以把这类真实请求形态固化进 CI，避免未来改 schema、adapter 或 WebUI 时不小心破坏协议边界。配合 `replay-conformance`，CI 还能覆盖本地 fake upstream 的 HTTP 路径；它仍不替代 live upstream 端到端测试。

### 33. 面向组织的策略分层

大型组织可以把策略分成几层：

- 全局安全策略。
- 团队默认策略。
- 项目策略。
- 模型策略。
- 单次请求策略。

合并顺序和冲突解决应由宿主网关声明，Gateway Harness 负责让每层策略都可验证、可解释。

### 34. 外部 GitOps 变更流

对高影响策略变更，可以走宿主系统之外的 GitOps/PR 流程：

- PR 修改 policy。
- CI 执行 `gateway-harness validate`。
- reviewer 检查用例影响。
- 合并后由网关热加载或滚动部署。

这不是 Gateway Harness 的底层 action，也不是网关内置审批状态机。它只是把上下文行为从“谁在后台随手改了 prompt”变成外部可审计的配置变更。

### 35. 与模型能力矩阵联动

不同模型能力不同：

- 是否支持工具调用。
- 是否支持 Responses。
- 是否支持长上下文。
- 是否支持视觉输入。
- 是否支持结构化输出。

Gateway Harness 可以把能力矩阵作为 condition 的输入来源之一；adapter 决定如何把实际模型能力同步进策略执行环境。

### 36. 开发、预发、生产环境分层

同一份网关在不同环境里可能需要不同策略：

- 开发环境开启更详细 trace。
- 预发环境启用新 hook。
- 生产环境只启用稳定 program。

Gateway Harness policy 可以按环境拆文件或拆 program；环境选择由部署系统或 adapter 注入。

### 37. 策略 dry-run

上线前可以先只模拟策略，不真正修改请求：

- 输出会命中哪些 program。
- 输出会执行哪些 action。
- 输出会在哪个 target 和 index 插入内容。
- 输出注入内容的 hash 和字符数，而不是原文。
- 输出哪些破坏性 action 被跳过。

这已经由 `gateway-harness dry-run-policy` 覆盖。它特别适合排查 `responses.compact.before_upstream` 这类压缩 hook：我们可以确认压缩事件会注入哪些提醒，同时保证 dry-run 不会偷偷执行 `context.truncate`，也不会因为 harness 自己估算 token 就制造额外上下文限制。

### 37.1 策略 apply + replay

dry-run 证明“计划会改什么”，但 adapter 还需要证明“实际改写后的请求仍然合法”。`replay-policy-conformance` 做的是本地闭环：

- 读取 conformance fixture。
- 校验 adapter capability、policy 和 request shape。
- 在内存里应用 policy 到请求副本。
- 把改写后的请求发到本地 fake upstream。
- 输出脱敏 trace、命中的 program、执行的 action、原始/改写后请求大小。

它不会打印完整改写请求，也不会请求真实模型。这样 CI 可以覆盖真实 apply 路径，同时不把网关日志变成 raw prompt/response 存储。

### 38. 在线问题诊断

当用户反馈“模型突然变啰嗦”或“上下文被改坏”时，可以通过 trace 回答：

- 哪个 policy 版本生效。
- 哪个 program 命中。
- 哪个 hook 修改了请求。
- 是否发生 truncate。
- 是否发生 failover 后的提示词注入。

这能把线上问题从猜测变成可排查事件。

### 39. 策略版本标记

每份 policy 可以带版本号：

- `version: "0.1"`
- `version: "2026-07-05"`
- `version: "team-a-2026w27"`

adapter 可以把 policy version 写入 trace，方便回溯某次请求到底跑的是哪一版上下文策略。

### 40. 用户组差异化体验

不同用户组可以有不同上下文体验：

- 免费组使用短提示词。
- 专业组使用完整提示词。
- 内部组启用实验性 hook。
- 管理员组启用更严格审计。

Gateway Harness 不负责鉴权，但可以让 adapter 根据用户组选择对应 program。

### 41. 模型迁移保护

从旧模型迁移到新模型时，可以临时注入迁移保护提示：

- 保持输出格式不变。
- 保持工具调用协议不变。
- 保持中文术语不变。
- 标记新模型和旧模型的行为差异。

迁移完成后移除这段 policy，比在业务代码里散落 prompt 更容易回滚。

### 42. 输出格式稳定化

某些业务依赖固定输出格式：

- JSON。
- Markdown。
- 表格。
- commit message。
- issue 模板。

Gateway Harness 可以在特定模型或 endpoint 上注入格式要求；真正的 schema 校验仍应由业务系统或 structured output 能力负责。

### 43. 防止提示词漂移

长期运行的网关容易出现提示词漂移：

- 不同渠道配置不一致。
- 不同模型默认 prompt 不一致。
- 临时补丁忘记撤销。
- UI 手动改动没有审计。

把策略放入 Git，并用 CLI/CI 验证，可以降低这类“隐形 prompt 配置”风险。

### 44. 多 adapter 兼容测试

同一份 policy 可能被多个 adapter 使用：

- NewAPI adapter。
- 本地 CLI adapter。
- 测试 harness adapter。
- 未来的 LiteLLM adapter。

每个 adapter 应声明支持的 hook/action 子集；policy 可以被验证为“对某个 adapter 兼容”。

### 45. 策略 lint

除了语法验证，还可以做 lint：

- program 名称是否清晰。
- hook 是否过宽。
- 注入文本是否过长。
- 是否缺少 reason。
- 是否使用 deprecated 字段。

v0.1 先做 validation；lint 可以作为后续 CLI 能力。

### 46. 策略影响面分析

改一条 policy 前，系统应能回答：

- 会影响哪些模型。
- 会影响哪些租户。
- 会影响哪些 endpoint。
- 会影响哪些 hook。
- 最坏会新增多少 token。

这类分析可以由 CLI 读取 policy 和 adapter capability 后生成报告。

### 47. 与计费系统协作

上下文注入会增加 prompt token，因此要和计费系统保持透明：

- trace 记录新增 token 估算。
- billing 可以区分用户原始输入和 harness 注入。
- 管理员能看到策略带来的成本变化。

Gateway Harness 只负责暴露可审计信息，不直接决定计费。

### 48. 与缓存策略协作

一些网关会做 prompt cache 或 channel affinity。

Gateway Harness 需要避免让无意义变化破坏缓存：

- 稳定注入文本。
- 避免随机时间戳。
- 避免每次请求生成不同 prompt。
- trace 里记录 hash 而不是原文。

如果 adapter 需要动态注入，应明确这会影响缓存命中。

### 49. 故障兜底策略

如果某个 action 执行失败，adapter 应有明确策略：

- fail closed：拒绝请求。
- fail open：跳过 harness 继续转发。
- degrade：只执行安全 action。

哪种策略正确取决于场景。安全策略更适合 fail closed，体验增强类策略更适合 fail open。

### 50. 最小可用 adapter

一个最小 adapter 不需要实现所有能力。

它可以先只支持：

- 读取 policy。
- 校验 hook/action。
- 在 `before_upstream` 注入 instructions。
- 输出脱敏 trace。

这样 Gateway Harness 可以从很小的集成开始，而不是一上来就要求完整可视化编程系统。

### 51. 项目/会话持久化审计

网关管理的会话应该能按项目划分，方便后续排查“这次上下文为什么这样变化”：

- 一个 project 对应一个仓库、业务项目或用户空间。
- 一个 session 对应一次连续任务、对话或工作流。
- event 记录 request、response、tool_call、compact、failover、harness_action、error。
- artifact 只保存 `content_hash` 和外部引用，不把原始 prompt/response 塞进账本。
- metadata 只放标签、ID 和脱敏状态，不放 `prompt`、`response` 或 `messages`。

这给“压缩时额外唤起 AI 总结”留出了正确位置：总结器产物可以作为 `compact_summary` artifact 被引用，但 Gateway Harness 核心不默认调用外部 AI。

### 52. AI-in-the-loop 上下文管家

更激进的方案是让一个 AI steward 参与上下文管理，但它必须是显式、可审计、可回滚的：

- 用户自然语言可以先生成 steward 或 policy 草案，但 Gateway Harness 只负责 schema validation；是否进入 PR/Console 审核由外部控制面决定。
- compact hook 可以唤起 steward 分析卡点，但输入应是 redacted trace、ledger metadata、artifact refs、session tags 和 user goal。
- steward 不能直接写任意 prompt，只能输出 `context.inject`、`ledger.artifact.create`、`diagnosis.note.create`、`session.tags.update` 等允许的结构化 action。
- steward proposal 必须再与 steward spec 交叉验证，未声明的 hook/action 一律不能执行。
- dry-run 只能模拟非破坏性输出，不能把 `context.truncate` 重新变成隐藏裁剪。
- `ledger.artifact.create` 必须要求 `artifact_hash_required`，避免把原始会话内容塞进网关日志。

这样可以实现“压缩时 AI 帮我提炼、诊断卡点、重新注入短提示”，但不会把网关变成一个看不见的 AI 黑盒。

如果 adapter 能用脱敏 token 统计发现“同一会话 key 的上下文断崖下降”，可以暴露
`context.continuity_drop.detected` hook，并配合 `when.context_continuity_drop=true` 显式触发短
Ledger 摘要注入。Gateway Harness 只校验和回放这个契约，不保存原始 prompt，不隐式调用 AI，也不能恢复客户端没有发送的完整历史。

## 不做什么

v0.1 明确不做：

- 不发布 patched NewAPI binary。
- 不发布 NewAPI Docker image。
- 不保存或 replay 上游 conversation state。
- 不在 ledger 契约里保存原始 prompt 或 response。
- 不伪造 Responses API 的 `item_reference`。
- 不执行任意脚本。
- 不定义隐式 program-level budget。
- 不把任何 harness 自己的 token 估算当作隐藏的模型上下文硬限制。
- 不在没有显式 steward spec 和 adapter 实现时偷偷调用 AI 管家。

这些边界是为了保证 Gateway Harness 先成为一个优雅、透明、可审计的策略契约，而不是变成另一个不可解释的代理层。

## 仓库结构

```text
cmd/gateway-harness/      CLI 入口
adapter/                  Adapter capability 类型和校验
conformance/              协议 fixture 校验
ledger/                   项目/会话审计账本校验
steward/                  AI-in-the-loop 上下文管家契约校验
policy/                   Policy 类型、校验和摘要
rule/                     归一化 Rule 类型、校验和 policy 编译
schema/                   JSON Schema
docs/                     概念和 adapter 契约
examples/newapi/          NewAPI adapter 示例 policy
fixtures/newapi/          NewAPI conformance fixtures
```

## 发布方式

Gateway Harness 独立发布。

主仓库发布：

- `gateway-harness` CLI。
- `gateway-harness.policy.schema.json`。
- `gateway-harness.rule.schema.json`。
- `gateway-harness.adapter.schema.json`。
- `gateway-harness.conformance.schema.json`。
- `gateway-harness.ledger.schema.json`。
- `gateway-harness.ledger-record.schema.json`。
- `gateway-harness.steward.schema.json`。
- `gateway-harness.steward-proposal.schema.json`。
- checksums。
- `gateway-harness-examples.tar.gz`，包含 examples、fixtures、docs 和 README。

网关特定补丁、adapter 实现、Docker 镜像或部署脚本，应放到独立 adapter 仓库或宿主网关仓库。

## 许可证

MIT
