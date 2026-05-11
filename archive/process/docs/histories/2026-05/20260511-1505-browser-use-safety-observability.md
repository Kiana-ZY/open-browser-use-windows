## [2026-05-11 15:05] | Task: Browser use safety observability

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `Codex Desktop / PowerShell / Windows`

### User Query

> 继续推进 Open Browser Use，让它成为 Codex app browser use 的平替甚至更优方案。

### Changes Overview

Scope: CLI runner, MCP server, agent docs, security docs, execution plan.

Key Actions:

- Added MCP direct-tool trace support so `obu mcp --trace-log` records the same JSONL action telemetry as `obu run --trace-log`.
- Kept `run_action_plan` trace output per action through the runner to avoid duplicate wrapper trace rows.
- Documented request id injection, risk labels, JSONL trace fields, and upper-runtime confirmation responsibilities.
- Marked M5 complete in the parity-plus execution plan and raised the observability quality score from C to B.

### Design Intent (Why)

M5 is meant to give agent runtimes a stable observability and policy input
surface without turning OBU itself into a Codex-specific policy engine. The
trace format links session, turn, action, risk, tab id, latency, and failure
state so Codex, Claude Code, shell workflows, and future SDK helpers can audit
browser work consistently.

### Files Modified

- `cmd/open-browser-use/mcp.go`
- `cmd/open-browser-use/mcp_test.go`
- `docs/ARCHITECTURE.md`
- `docs/CODEX_AND_CLAUDE_USAGE.md`
- `docs/SECURITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/exec-plans/active/2026-05-11-browser-use-parity-plus.md`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `skills/open-browser-use/references/sdk-and-protocol.md`

### Validation

- `go test ./...`
- `go build -o open-browser-use.exe ./cmd/open-browser-use`
- Real action-plan smoke with `run --trace-log`: opened `about:blank`, read
  `page-info`, finalized tabs, verified trace rows and empty session tabs.
- Real MCP smoke with `mcp --trace-log`: called `ping`, verified structured MCP
  result and a JSONL trace row.
