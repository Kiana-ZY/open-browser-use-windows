## [2026-05-11 14:33] | Task: browser use interaction reliability

### Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### User Query

> 继续推进 Browser Use parity-plus 计划。

### Changes Overview

**Scope:** CLI / action runner / MCP / agent docs / execution plan

**Key Actions:**

- **[Interaction]**: Reworked `click` to scroll the target into view, check
  disabled and visible states, dispatch pointer / mouse events, and return a
  structured action result.
- **[Interaction]**: Reworked `fill` to support input, textarea, select, and
  contenteditable targets with input/change events and structured diagnostics.
- **[Runner]**: Added `click` and `fill` action-plan steps that reuse the same
  interaction helper as direct CLI commands.
- **[MCP]**: Added `click` and `fill` tools with `structuredContent` results.
- **[Tests]**: Added Go coverage for runner and MCP interaction results.
- **[Docs]**: Updated skill and CLI README to describe click/fill JSON status
  and failure reasons.

### Design Intent (Why)

The previous implementation used a thin DOM `.click()` / `.value = ...` path,
which was fast but fragile on modern web apps. This keeps the light DOM-first
approach while adding enough browser-like event sequencing and diagnostics for
agents to recover intelligently.

### Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `cmd/open-browser-use/mcp.go`
- `cmd/open-browser-use/mcp_test.go`
- `docs/exec-plans/active/2026-05-11-browser-use-parity-plus.md`
- `docs/histories/2026-05/20260511-1433-browser-use-interaction-reliability.md`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
