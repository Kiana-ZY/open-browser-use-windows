# Codex And Claude Code Usage

这份文档专门说明怎么把 Open Browser Use 作为 skill + MCP 工具接进
Codex 和 Claude Code，并尽量让两边都走同一套稳定、低歧义的 agent-facing
contract。

## 适用场景

优先在这些场景使用 OBU：

- 需要接管用户真实 Chrome / Edge 标签页，而不是无状态无登录浏览器。
- 需要复用用户已有 cookie、登录态、历史记录、下载、文件选择器或 tab group。
- 需要在 shell-first 或 MCP-first 的 agent runtime 里保持轻量接入。

更适合用内建 browser use 的场景：

- 任务只发生在 Codex app 的 in-app browser 里。
- 你更需要 runtime 已内置的安全边界、交互策略和 UI 集成，而不是跨 runtime
  复用。

## Skill 安装

Codex:

```text
~/.codex/skills/open-browser-use/
```

Claude Code:

```text
.claude/skills/open-browser-use/
```

把仓库里的 `skills/open-browser-use/` 整个复制过去即可。skill 本身负责告诉
agent：

- 诊断不确定或连接失败时先 `obu doctor --browser all --json`
- 日常任务启动时先 `obu ping`
- 再创建唯一 session id
- 再 `open-tab` / `claim-tab`
- 再做 `page-info` / `text` / `snapshot` / `cdp`
- 最后 `finalize-tabs`

## Codex 接法

### MCP 配置

把下面配置加入 `~/.codex/config.toml`：

```toml
[mcp_servers.open_browser_use]
command = "obu"
args = ["mcp", "--session-id", "obu-<task-or-thread-id>"]
```

如果 runtime 会为每轮任务注入唯一 id，优先把它直接塞进 `--session-id`。不要
在 agent 工作流里长期依赖默认 `obu-cli` session。

只有在外层 runtime 已经明确给了 socket 时，才额外传：

```toml
args = ["mcp", "--session-id", "obu-<task-id>", "--socket", "/tmp/open-browser-use/example.sock"]
```

需要审计 agent 浏览器动作时，加上：

```toml
args = ["mcp", "--session-id", "obu-<task-id>", "--trace-log", "/tmp/obu-trace.jsonl"]
```

`run_action_plan` 和直接 MCP tool call 都会写 JSONL trace，包含 session、turn、
action、risk、tab id、耗时和成功/失败状态。

### 推荐工作流

1. 环境或连接状态不确定时，先让 agent 调 `doctor`，参数用 `browser: "all"`
2. 日常任务启动时调 `ping`
3. 再调 `name_session`
4. 再 `open_tab` 或 `claim_tab`
5. 用 `page_info` / `history` / `cdp` / `run_action_plan`
6. 结束时统一 `finalize_tabs`

### 推荐提示词

```text
用 $open-browser-use 打开页面并读取 page_info，最后 finalize tabs。
```

```text
用 OBU claim 当前用户的 GitHub tab，抽取正文，保留一个 handoff tab。
```

## Claude Code 接法

Claude Code 更常见的是 skill-first + shell-first 两种接法。

### 方式一：直接走 skill + CLI

让 Claude Code 按 skill 里的启动顺序调用：

```text
先用 open-browser-use skill 做 doctor/ping，然后开新 session，打开 https://example.com，读 page-info，最后 finalize-tabs。
```

这种方式最稳，适合单轮、短链路、命令可见的工作。

### 方式二：把 OBU 作为本地 MCP server

如果 Claude Code 运行环境支持本地 MCP server，也可以复用同样的：

```text
command = obu
args = mcp --session-id obu-<task-id>
```

关键建议和 Codex 一样：

- 每个任务一个 session id
- 优先用稳定高层工具，不先上 unrestricted `call`
- 最后一条浏览器动作是 `finalize_tabs`

## 稳定对象协议

为了让 Codex、Claude Code、CLI、MCP、skill 文档说的是同一件事，常用路径推荐
统一消费下面这些对象形状。

CLI `--json` 与 MCP `structuredContent` 都应尽量对齐到：

- `doctor` -> 单浏览器 `{ "ok": ..., "socket": ..., "nativeHost": ..., "browserExtension": ..., "checks": [...] }`
- `doctor browser=all` -> `{ "ok": ..., "browsers": [...], "nextSteps": [...] }`
- `ping` -> `{ "status": "pong" }`
- `open_tab` / `open-tab` -> `{ "tab": ..., "navigate": ...? }`
- `claim_tab` / `claim-tab` -> `{ "tab": ... }`
- `navigate` -> `{ "navigate": ... }`
- `page_info` / `page-info` -> `{ "title": ..., "url": ..., "readyState": ..., "text": ... }`
- `text` -> `{ "text": ... }`
- `snapshot` -> `{ "items": [...] }`
- `screenshot` -> `{ "path": ..., "bytes": ..., "format": "png", "clip": ...? }`
- `click` / `fill` -> `{ "ok": true, "action": ..., "ref": ... }`，失败时带 `reason`
- `history` -> `{ "items": [...] }`
- `tabs` / `user_tabs` -> `{ "items": [...] }`
- `wait_load` / `wait` -> `{ "readyState": ... }`
- `name_session` / `finalize_tabs` / `turn_ended` -> `{ "ok": true }`

这样 agent 可以少写很多“先判断是不是裸数组、是不是包在 result 里、是不是还要
拆 CDP wrapper”的胶水逻辑。`run_action_plan` / `open-browser-use run` 的顶层
输出还包含 `ok`；失败时仍返回已经完成的 steps，并在失败 step 上写入 `ok:
false` 与 `error`。需要继续采集后续诊断证据时，CLI 使用
`--continue-on-error`，MCP 使用 `continue_on_error: true`。

## 多模态与 token 预算

GPT-5 级多模态模型应该把截图当成视觉关键帧，而不是主要数据通道。推荐顺序是：

- 用 `page-info --max-chars 1200` 或 `text --selector main --max-chars 2000` 获取低 token 文本。
- 只有要点击/填写时再跑 `snapshot --limit 20` 获取 refs。
- 在导航后、关键交互前后或失败时保存一张 viewport / selector screenshot。

截图命令只返回本地文件路径、字节数和格式，不把 base64 写进 stdout 或 trace。
这样模型仍能按需看图，同时 CLI/MCP 的结构化输出不会被图片数据撑爆。
再拆一层”的适配逻辑。

## 安全与可观测

OBU 会给没有显式 `request_id` 的浏览器 RPC 自动补一个 request id。action
plan 和带 `--trace-log` 的 MCP direct tool call 还会给动作打 risk 标签：

- `read`：doctor、读取状态、tab、history、page-info、text、snapshot、screenshot。
- `navigation`：打开、认领、导航 tab。
- `interaction`：click、fill、move-mouse。
- `file-system`：设置 file chooser 文件。
- `session`：name-session、turn-ended、finalize-tabs。
- `unrestricted`：cdp、call。

这些标签用于审计和上层策略判断，不等于 OBU 自己会拦截。Codex / Claude
Code / 自定义 runtime 应该在上传、下载、剪贴板、提交表单、购买、删除、发送外部消息
以及可能有副作用的 unrestricted 调用前做用户确认。

## 选择建议

Codex app 自带 browser use 的优势：

- 和应用内 tab、界面、交互回路结合更深。
- 对本地网页、in-app browser、并排验证更顺手。
- 对 Codex 自身产品路径来说，默认体验更一致。

Open Browser Use 的优势：

- 能跨 Codex / Claude Code / shell / SDK / MCP 复用一套真实浏览器能力。
- 接的是用户真实浏览器 profile，不是新开无状态环境。
- CLI、MCP、JS/Python/Go SDK 都能共享同一底层。

比较合适的默认策略是：

- Codex app 内本地页面和并排验证，优先内建 browser use。
- 跨 runtime、要真实登录态、要沉淀统一 skill / MCP contract，优先 OBU。
