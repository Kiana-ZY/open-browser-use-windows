# Browser Use Parity Plus

## 目标

把 Open Browser Use 打造成 Codex app 内置 Browser Use 的平替，并在真实
Chrome / Edge profile、跨 runtime 接入、低 token 浏览器读写和 agent 可脚本化
方面形成更强优势。最终状态不是只补齐几个命令，而是形成一套稳定的
agent-facing browser control contract：CLI、MCP、SDK、skill 和文档说同一套
对象、同一套动作、同一套安全边界。

## 范围

- 包含：
  - CLI / action runner / MCP 的高频浏览动作统一。
  - 页面读取、元素快照、截图、点击、输入、等待、tab 生命周期的稳定 JSON 契约。
  - token-aware extraction：支持 bounded text、selector text、bounded snapshot。
  - 真实浏览器 smoke：GitHub Trending 这类公开页面、本地页面、长页面和链接密集页。
  - Agent skill 和使用文档同步更新。
  - 安全与权限策略的渐进式设计：敏感动作二次确认留给上层 runtime，但 OBU 提供可观测 hooks。
- 不包含：
  - 复刻 Codex app 私有 UI 或私有 runtime 策略。
  - 绕过网站风控、验证码或未授权访问。
  - 一次性重写 extension / native host / SDK 全部架构。

## 背景

- 相关文档：
  - `docs/ARCHITECTURE.md`
  - `docs/CODEX_AND_CLAUDE_USAGE.md`
  - `docs/wiki/browser-client/notes/open-browser-use-vs-codex-browser-use.md`
  - `skills/open-browser-use/SKILL.md`
- 相关代码路径：
  - `cmd/open-browser-use/main.go`
  - `cmd/open-browser-use/mcp.go`
  - `cmd/open-browser-use/main_test.go`
  - `cmd/open-browser-use/mcp_test.go`
  - `apps/chrome-extension/background.js`
  - `packages/open-browser-use-js/`
  - `packages/open-browser-use-python/`
  - `packages/open-browser-use-go/`
- 已知约束：
  - Windows fork 使用 TCP `127.0.0.1:19832`，macOS / Linux 仍保留 Unix socket
    架构语义。
  - 当前 OBU 直接接管用户真实浏览器，不内置 Codex 风格站点策略。
  - 上层 agent 必须使用唯一 session id，并在结束时 finalize tabs。

## 风险

- 风险：命令名、JSON shape、MCP 工具名继续漂移，agent 集成成本升高。
  - 缓解方式：先定义稳定对象协议，并用 CLI 与 MCP 测试守住。
- 风险：为了追求能力补齐而把 CLI 做成大而脆的半个 Playwright。
  - 缓解方式：保持 high-level agent actions 小而正交，复杂逃生口继续使用 CDP / unrestricted call。
- 风险：真实浏览器控制权限太大，被不可信上层滥用。
  - 缓解方式：把敏感动作分类、审计点和确认建议写进文档；后续引入 policy hooks。
- 风险：bounded extraction 做得太粗，损失页面信息。
  - 缓解方式：默认低 token，参数允许 agent 明确扩大范围。

## 里程碑

1. M1 - Agent-facing contract 收敛
   - 统一 CLI、runner、MCP 高频动作命名。
   - `page-info`、`text`、`snapshot` 在直接 CLI 与 runner 中都可用。
   - `--json` 输出保持对象形状，不让 agent 猜包装层。
2. M2 - Token-aware extraction
   - `text --max-chars --selector`。
   - `page-info --max-chars --selector`。
   - `snapshot --limit`，runner 也支持等价参数。
3. M3 - Interaction reliability
   - 给 `click` / `fill` 增加可见性、滚动、事件序列和失败诊断。
   - 形成轻量 DOM first、CDP fallback second 的操作梯。
4. M4 - Visual verification
   - 截图输出、元素截图、页面可视状态摘要。
   - 增加本地 smoke 脚本验证截图非空、页面 ready、元素可读。
5. M5 - Safety and observability
   - request id、动作日志、敏感动作标签、policy hook 设计。
   - 文档说明哪些动作需要上层确认。
6. M6 - SDK / MCP parity
   - JS / Python / Go SDK 对齐核心对象协议。
   - MCP tools 与 CLI 参数能力对齐。

## 验证方式

- 命令：
  - `go test ./...`
  - `pnpm test`
  - `python -m pytest packages/open-browser-use-python`
  - `./scripts/ci.sh`
- 手工检查：
  - `open-browser-use ping --session-id <id>`
  - `open-browser-use run --session-id <id> -c "<open/wait/page/text/snapshot/finalize>"`
  - GitHub Trending 页面读取：确认仓库列表、链接、stars/forks 和今日增长可抽取。
  - 长页面读取：确认 bounded extraction 不爆上下文。
- 观测检查：
  - 每个浏览动作返回稳定 JSON。
  - 失败时错误信息包含 action、tab id、建议下一步。
  - 结束时 session tabs 被清理或按 deliverable/handoff 保留。

## 进度记录

- [x] 2026-05-11：完成本地体验评估，确认 OBU 能读取 GitHub Trending 和仓库详情页。
- [x] 2026-05-11：确认主要短板是 contract 漂移、bounded extraction 缺失、交互层偏薄。
- [x] M1：对齐 CLI、runner、MCP 高频动作。
- [x] M2：加入 bounded text / page-info / snapshot。
- [x] M3：补强 click / fill 可靠性。
- [x] M4：补视觉验证能力。
- [x] M5：补安全与可观测策略。
- [x] M6：对齐 SDK / MCP parity。

## 决策记录

- 2026-05-11：优先打造真实浏览器路线的强项，而不是逐项复刻 Codex app 内置
  Browser Use。OBU 的优势应是用户真实 profile、跨 runtime、低 token、可脚本化。
- 2026-05-11：第一阶段从 CLI / runner / MCP contract 开始，因为这是 agent
  成功率的地基，也是后续 SDK 和 skill 的共同语言。
- 2026-05-11：把 `text` / `snapshot` 纳入 action runner 和 MCP，高频读取路径
  不再需要在 runner、直接 CLI、MCP 之间切换；同时让 `text` / `page-info`
  支持 `selector` 和 `max-chars`，让 `snapshot` 支持 `limit`。
- 2026-05-11：把 `click` / `fill` 升级为可诊断交互动作：先滚动到可见、检查
  disabled / visibility，再派发 pointer / mouse / input / change 事件，并返回
  `{ok, action, ref, tag, text, rect}` 或失败 reason。MCP 与 action runner 复用
  同一套实现。
- 2026-05-11：把 `screenshot` 升级为结构化视觉验证动作：直接 CLI、action
  runner 和 MCP 都返回 `{path, bytes, format, tabId, selector?, clip?}`，并支持
  viewport、`--selector` 元素截图和 `--full-page` 整页截图。
- 2026-05-11：M5 先做观测而不是内建强制 policy。CLI / runner / MCP 对
  browser RPC 自动补 `request_id`；`run --trace-log` 和 `mcp --trace-log`
  写 JSONL trace；动作按 `read`、`navigation`、`interaction`、
  `file-system`、`session`、`unrestricted` 分类，供 Codex、Claude Code 或
  其他上层 runtime 做确认策略。
- 2026-05-11：M6 采用“薄 SDK helper”而不是引入独立浏览器引擎。JS、
  Python、Go SDK 都新增和 CLI/MCP 对齐的结构化 helper：page info、text、
  snapshot、screenshot、click、fill；已有 `domSnapshot` / locator 字符串
  helper 保持兼容。
