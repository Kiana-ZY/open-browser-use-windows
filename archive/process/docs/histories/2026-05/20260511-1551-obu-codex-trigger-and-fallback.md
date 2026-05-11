## [2026-05-11 15:51] | Task: OBU Codex trigger and fallback

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `Codex Desktop / PowerShell / Windows`

### User Query

> OBU 的 token 消耗如果不多，就帮我改成类似 `@chrome` 能触发，并在 Codex Browser 启动不了时可用。

### Changes Overview

Scope: agent skill metadata, Codex local config, repository agent instructions.

Key Actions:

- Updated the Open Browser Use skill description to trigger on `@obu`,
  `@open-browser-use`, `@real-browser`, and Browser fallback requests.
- Added repository `AGENTS.md` fallback rules for Web/frontend checks when the
  Codex app built-in Browser cannot start or connect.
- Documented low-token OBU defaults: bounded `page-info`, selector text,
  bounded snapshot, and JSON screenshot metadata.
- Synced the skill into the local Codex skill directory.
- Added a Codex MCP server entry for `open_browser_use` with trace logging.

### Design Intent (Why)

OBU is suitable as a low-token browser fallback when agents use bounded
extraction instead of whole-page DOM dumps. The trigger should not steal the
official `@chrome` namespace, so the stable user-facing trigger is `@obu`.

### Files Modified

- `AGENTS.md`
- `skills/open-browser-use/SKILL.md`
- `C:\Users\yuhua\.codex\skills\open-browser-use\SKILL.md`
- `C:\Users\yuhua\.codex\skills\open-browser-use\references\*.md`
- `C:\Users\yuhua\.codex\config.toml`

### Validation

- Confirmed repository and local Codex skill frontmatter include `@obu`.
- Confirmed `open_browser_use` MCP entry exists in Codex config.
- Ran `open-browser-use.exe ping --session-id obu-config-smoke --json` and got
  `{ "status": "pong" }`.
