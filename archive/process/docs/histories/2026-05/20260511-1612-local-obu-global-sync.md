## [2026-05-11 16:12] | Task: Local OBU global sync

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `Codex Desktop / PowerShell / Windows`

### User Query

> Update the global Codex, Claude Code, and local Open Browser Use setup, but use
> the locally built project version rather than the published npm package.

### Changes Overview

Scope: local build artifact, user-level CLI shims, Codex config, Claude Code
MCP config, agent skills, browser native messaging manifest.

Key Actions:

- Rebuilt the local Windows CLI binary from the repository.
- Re-linked the user-level npm package entry to the repository package.
- Replaced the user-level `open-browser-use.exe` and `obu.exe` shims with the
  locally built repository binary.
- Synced the repository `open-browser-use` skill to Codex, Claude Code, and the
  shared agents skill directory.
- Pointed Codex and Claude Code MCP server configs at the repository binary.
- Updated Chrome and Edge native messaging host manifests to launch the
  repository binary directly.

### Design Intent (Why)

The global environment briefly resolved to the latest published npm package,
but the desired development setup is to exercise the local working tree. Keeping
Codex, Claude Code, PATH commands, and browser native messaging on the same
repository binary prevents version drift while testing new local changes.

### Files Modified

- `docs/histories/2026-05/20260511-1612-local-obu-global-sync.md`
- `C:\Users\yuhua\.codex\config.toml`
- `C:\Users\yuhua\.codex\skills\open-browser-use\*`
- `C:\Users\yuhua\.claude.json`
- `C:\Users\yuhua\.claude\skills\open-browser-use\*`
- `C:\Users\yuhua\.agents\skills\open-browser-use\*`
- Chrome and Edge native messaging host manifests

### Validation

- `open-browser-use version` and `obu version` both returned the local project
  version.
- PATH `open-browser-use.exe`, PATH `obu.exe`, and the repository binary had the
  same SHA-256 hash.
- `open-browser-use ping --session-id obu-local-final --json` returned
  `{ "status": "pong" }`.
- `open-browser-use info --json` returned the connected browser extension.
- The active browser native host process launched from the repository binary.
- `claude mcp list` and `claude mcp get open_browser_use` reported the
  repository binary MCP server as connected.
- `go test ./cmd/open-browser-use ./internal/host ./internal/wire` passed.
