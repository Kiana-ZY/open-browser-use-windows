## [2026-05-11 12:59] | Task: align CLI page-info surface

### 🤖 Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### 📥 User Query

> 继续

### 🛠 Changes Overview

**Scope:** `cmd/open-browser-use`, `packages/open-browser-use-cli`, `skills/open-browser-use`, `docs/histories`

**Key Actions:**

- **[CLI]**: Added a direct `page-info` subcommand so the CLI matches the existing action-runner and MCP concept for page summary reads.
- **[Test]**: Added a command-level test that verifies `page-info --json` returns the raw summary object rather than a wrapped JSON-RPC envelope.
- **[Docs]**: Updated the CLI README and repo-local OBU skill to distinguish `page-info` for compact summary reads from `text` for raw body text.

### 🧠 Design Intent (Why)

The repository already exposed `page_info` through MCP and `page-info` through
the action runner, but not as a direct CLI command. That made the same concept
appear under different names depending on which surface an agent used. Adding a
direct CLI `page-info` command is a low-risk way to unify the agent-facing
mental model without removing the lightweight `text` path.

### 📁 Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `docs/histories/2026-05/20260511-1259-cli-page-info-alignment.md`
