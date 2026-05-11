## [2026-05-11 18:36] | Task: fix GitHub Go CI output handling

### Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop on Windows / PowerShell + WSL bash`

### User Query

> GitHub running Go failed.

### Changes Overview

**Scope:** Go CLI command output and MCP test reliability.

**Key Actions:**

- **[CLI]**: Routed Cobra command JSON and text output through `cmd.OutOrStdout()` so tests and embedding callers can capture output instead of receiving empty buffers.
- **[Helper]**: Added `invokeAndWriteTo` while keeping the existing stdout-based helper for compatibility.
- **[Test]**: Made the MCP screenshot mock socket server close-driven instead of waiting for a fixed number of accepts, avoiding CI timeouts when the command needs fewer browser RPCs.
- **[Verification]**: Ran targeted Go tests, `go test ./... -count=1`, and the repository CI script locally.

### Design Intent (Why)

GitHub Actions failed because several CLI handlers wrote JSON to the process
stdout directly while tests had redirected Cobra output to a buffer. The MCP
screenshot test also assumed a fixed socket accept count and could block after
the server had already completed the useful work. Respecting Cobra's writer
contract and closing the mock listener after the MCP server finishes makes the
tests deterministic in CI and preserves normal CLI stdout behavior.

### Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/mcp_test.go`
- `archive/process/docs/histories/2026-05/20260511-1836-github-go-ci-output-fix.md`
