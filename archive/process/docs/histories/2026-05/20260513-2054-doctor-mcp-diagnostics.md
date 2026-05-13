## [2026-05-13 20:54] | Task: Expose doctor diagnostics through MCP

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `Codex Desktop / PowerShell / Windows`

### User Query

> 继续

### Changes Overview

Scope: CLI diagnostics, MCP server, agent docs, reliability/security docs.

Key Actions:

- Added a `doctor` MCP tool with optional `browser` argument and
  `structuredContent` matching CLI `doctor --json`.
- Added `doctor --browser all --json` and MCP `browser: "all"` suite reports
  for Chrome/Edge preflight diagnostics.
- Reused the existing `buildDoctorReport` path so CLI and MCP diagnostics stay
  aligned.
- Classified traced MCP `doctor` calls as `read`.
- Added MCP tests for tool listing, cross-platform diagnostic failure reports,
  and Unix fake-relay success reports.
- Updated Codex/Claude usage, npm CLI README, bundled skill, architecture, and
  security docs to list `doctor` as the first setup/connectivity diagnostic.
- Added a CloakBrowser comparison document that borrows product workflow,
  diagnostics, release, and troubleshooting patterns while explicitly avoiding
  stealth, CAPTCHA-bypass, and fingerprint-spoofing scope.

### Design Intent (Why)

Agents should not have to shell out to the CLI just to understand why browser
automation is unavailable. Exposing the same diagnostic object through MCP lets
MCP-first runtimes inspect native host, manifest, relay, extension, and
next-step state directly while preserving the CLI contract.

### Files Modified

- `cmd/open-browser-use/mcp.go`
- `cmd/open-browser-use/mcp_test.go`
- `cmd/open-browser-use/main.go`
- `docs/ARCHITECTURE.md`
- `docs/CLOAKBROWSER_COMPARISON.md`
- `docs/CODEX_AND_CLAUDE_USAGE.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- `docs/releases/feature-release-notes.md`
- `archive/process/docs/QUALITY_SCORE.md`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `skills/open-browser-use/references/troubleshooting.md`
- `skills/open-browser-use/references/sdk-and-protocol.md`

### Validation

- `go test ./cmd/open-browser-use`
- `go test ./...`
- `bash ./scripts/check-docs.sh`
- `bash ./scripts/check-repo-hygiene.sh`
