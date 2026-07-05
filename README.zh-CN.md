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
- `Action`：策略要做什么，例如 `context.inject` 或 `context.truncate`。
- `Condition`：策略什么时候生效，例如匹配模型或 token 估算。
- `Budget`：只限制 Gateway Harness 自己引入的改动，不伪造模型上下文窗口。
- `Trace`：记录脱敏审计信息，例如命中的 program、hook、操作数和内容 hash。
- `Adapter`：宿主网关的胶水层，例如 NewAPI adapter。

## v0.1 已支持什么

v0.1 是一个最小但可发布的契约版本：

- Go policy structs。
- Policy validation。
- JSON Schema。
- CLI：`validate`、`explain`、`schema`。
- NewAPI adapter contract 文档。
- NewAPI 示例 policy。
- Release 产物：Linux amd64、Linux arm64、Linux armv7、Windows amd64。

当前内置 action：

- `context.inject`：注入一段上下文。
- `context.truncate`：保留最近若干条上下文，并可保留指定角色。

当前内置 hook：

- `request.before_model_mapping`
- `request.before_upstream`
- `chat.before_model_mapping`
- `chat.before_upstream`
- `responses.before_model_mapping`
- `responses.before_upstream`
- `responses.compact.before_model_mapping`
- `responses.compact.before_upstream`

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

## 示例 policy

```json
{
  "version": "0.1",
  "programs": [
    {
      "name": "coding-model-harness",
      "models": ["gpt-5.4-mini", "kimi-for-coding"],
      "tags": ["domain:coding"],
      "budget": {
        "max_patch_ops": 16,
        "max_added_tokens": 1200
      },
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
- 负数预算。

### 9. 生成脱敏 trace

adapter 执行策略时应写入脱敏 trace：

- 命中的 program。
- 执行 hook。
- 操作数量。
- 新增 token 估算。
- 注入内容 hash。

不要把完整提示词直接写入日志，避免把用户数据或内部 prompt 泄漏出去。

### 10. 限制 harness 自己的 blast radius

`budget.max_added_tokens` 和 `budget.max_patch_ops` 用来限制 Gateway Harness 自己能做多少改动。

它们不应该被解释成模型的最大上下文窗口。Gateway Harness 不应该因为自己的估算比上游模型小，就拒绝用户请求。

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
- 降低 `max_added_tokens`。

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

### 33. 面向组织的策略分层

大型组织可以把策略分成几层：

- 全局安全策略。
- 团队默认策略。
- 项目策略。
- 模型策略。
- 单次请求策略。

合并顺序和冲突解决应由宿主网关声明，Gateway Harness 负责让每层策略都可验证、可解释。

### 34. 人工审批流

对高影响策略变更，可以走人工审批：

- PR 修改 policy。
- CI 执行 `gateway-harness validate`。
- reviewer 检查用例影响。
- 合并后由网关热加载或滚动部署。

这样上下文行为不再是“谁在后台随手改了 prompt”，而是可审计的配置变更。

### 35. 与模型能力矩阵联动

不同模型能力不同：

- 是否支持工具调用。
- 是否支持 Responses。
- 是否支持长上下文。
- 是否支持视觉输入。
- 是否支持结构化输出。

Gateway Harness 可以把能力矩阵作为 condition 的输入来源之一；adapter 决定如何把实际模型能力同步进策略执行环境。

## 不做什么

v0.1 明确不做：

- 不发布 patched NewAPI binary。
- 不发布 NewAPI Docker image。
- 不保存或 replay 上游 conversation state。
- 不伪造 Responses API 的 `item_reference`。
- 不执行任意脚本。
- 不把 `max_context_tokens` 当作隐藏的模型上下文硬限制。

这些边界是为了保证 Gateway Harness 先成为一个优雅、透明、可审计的策略契约，而不是变成另一个不可解释的代理层。

## 仓库结构

```text
cmd/gateway-harness/      CLI 入口
policy/                   Policy 类型、校验和摘要
schema/                   JSON Schema
docs/                     概念和 adapter 契约
examples/newapi/          NewAPI adapter 示例 policy
```

## 发布方式

Gateway Harness 独立发布。

主仓库发布：

- `gateway-harness` CLI。
- `gateway-harness.policy.schema.json`。
- checksums。
- 示例 policy。

网关特定补丁、adapter 实现、Docker 镜像或部署脚本，应放到独立 adapter 仓库或宿主网关仓库。

## 许可证

MIT
