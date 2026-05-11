## [2026-05-11 15:00] | Task: browser use parity plus m1

### Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### User Query

> 先写计划，再开始，把 Open Browser Use 打造成 Codex app 自带 Browser Use
> 的平替，甚至在真实浏览器路线下更优秀。

### Changes Overview

**Scope:** CLI / action runner / MCP / agent docs / execution plan

**Key Actions:**

- **[Plan]**: Added an active parity-plus execution plan that defines the
  long-running roadmap, risks, milestones, and validation strategy.
- **[CLI]**: Added bounded extraction options: `text --selector --max-chars`,
  `page-info --selector --max-chars`, and `snapshot --limit`.
- **[Runner]**: Added `text` and `snapshot` action-plan steps so common read
  workflows no longer need to leave `open-browser-use run`.
- **[MCP]**: Added `text` and `snapshot` tools and passed bounded extraction
  arguments through `page_info`.
- **[Tests]**: Added regression coverage for action-plan `text` / `snapshot`
  and MCP tool listing/calls.
- **[Docs]**: Updated the repo skill and npm CLI README with the new bounded
  read paths.

### Design Intent (Why)

The first product gap was not raw browser power; it was drift between CLI,
action runner, MCP, and skill guidance. This change makes the common page-read
path consistent and token-aware before deeper interaction or visual work starts.

### Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `cmd/open-browser-use/mcp.go`
- `cmd/open-browser-use/mcp_test.go`
- `docs/exec-plans/active/2026-05-11-browser-use-parity-plus.md`
- `docs/histories/2026-05/20260511-1500-browser-use-parity-plus-m1.md`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
