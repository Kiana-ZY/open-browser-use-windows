# Architecture

Open Browser Use for Windows is a Windows-first browser automation bridge for
local agent runtimes. It connects a Chromium extension, a Go native messaging
host, a local client relay, and CLI/SDK/MCP clients so agents can operate the
user's real Chrome or Microsoft Edge profile.

The short command remains `obu`; the full binary and package command remain
`open-browser-use`.

## Runtime Topology

```text
Chrome / Microsoft Edge MV3 extension
  -> Chromium Native Messaging stdio
  -> open-browser-use.exe
  -> Windows TCP relay on 127.0.0.1:19832
  -> CLI / MCP server / JavaScript SDK / Python SDK / Go SDK
```

The same command surface is used by shell-first agents, MCP clients, and SDK
users. Common read paths should prefer bounded calls such as `page-info`,
`text`, `snapshot`, and `screenshot` instead of dumping full DOM payloads.

## Compatibility Identifiers

The browser native messaging host name is intentionally still:

```text
com.ifuryst.open_browser_use.extension
```

Chrome native messaging host names are part of the installed extension/host
contract and only allow lowercase letters, digits, underscores, and dots.
Changing this identifier would break existing manifests unless a migration path
is added. User-facing project names and repository URLs now use Open Browser
Use for Windows, but this protocol identifier stays stable for compatibility.

## Repository Structure

- `apps/chrome-extension/`: MV3 extension for Chrome and Microsoft Edge. It
  owns tab/session state, CDP access, cursor overlay injection, downloads,
  clipboard helpers, file chooser handling, and popup status display.
- `cmd/open-browser-use/`: Go CLI, Chromium native messaging entrypoint, setup
  helpers, action runner, and stdio MCP server.
- `internal/host/`: Native host relay and platform transport. Windows uses the
  TCP relay; Unix-like paths remain for cross-platform build compatibility.
- `internal/wire/`: Native messaging frame codec.
- `packages/open-browser-use-cli/`: npm binary package exposing
  `open-browser-use` and `obu`.
- `packages/open-browser-use-js/`: JavaScript/TypeScript SDK distributed as
  `open-browser-use-sdk`.
- `packages/open-browser-use-python/`: Python SDK distributed as
  `open-browser-use-sdk`, with import module `open_browser_use`.
- `packages/open-browser-use-go/`: Go SDK import path
  `github.com/Kiana-ZY/open-browser-use-windows/packages/open-browser-use-go`.
- `skills/open-browser-use/`: Agent-facing operating guide for Codex, Claude
  Code, and other shell-first runtimes.
- `docs/`: Active project documentation that must stay aligned with current
  runtime behavior.
- `archive/`: Process history, plans, research snapshots, old docs, and local
  agent snapshots that are useful for provenance but not runtime dependencies.
- `scripts/`: Build, packaging, release, CI, documentation, and repository
  hygiene automation.

## Extension Responsibilities

The extension exposes Browser Use-style JSON-RPC handlers through native
messaging, including:

- `getInfo`, `createTab`, `getTabs`, `getUserTabs`, `getUserHistory`
- `claimUserTab`, `finalizeTabs`, `nameSession`, `turnEnded`
- `attach`, `detach`, `executeCdp`
- `moveMouse`, cursor state forwarding, and cursor arrival notifications
- `waitForFileChooser`, `setFileChooserFiles`
- `waitForDownload`, `downloadPath`
- clipboard read/write helpers

Session state is stored in `chrome.storage.local` so MV3 service worker restarts
can recover tab group ownership. New sessions default to `Task - OBU`, while
final deliverables move into the shared `✅ Open Browser Use` group.

## CLI And MCP Responsibilities

The CLI provides setup, diagnostics, direct browser commands, JSON output,
line-oriented action plans, and an MCP server:

- `setup`, `setup beta`, `doctor`, `install-manifest`, and `manifest` manage
  browser integration and diagnostics.
- `ping`, `info`, `user-tabs`, `open-tab`, `claim-tab`, `navigate`,
  `page-info`, `text`, `snapshot`, `click`, `fill`, `screenshot`, `history`,
  `cdp`, and `finalize-tabs` cover common agent workflows.
- `run` executes small action plans with shared session/tab context.
- `mcp` exposes the same surface through stdio JSON-RPC for agent runtimes.

CLI action plans and MCP direct tool calls can write JSONL trace rows with
session, turn, action, risk class, tab id, duration, and status. The risk labels
are observability hints for upper-layer runtimes, not an authorization boundary.

## SDK Responsibilities

The JavaScript, Python, and Go SDKs keep low-level request access available
while adding high-level browser/tab helpers. SDKs should preserve the same
structured result shapes as CLI `--json` and MCP `structuredContent` where
practical.

SDKs do not implement Codex-style site policies, user approval gates, or
destructive-action confirmation. Integrations that accept remote or untrusted
instructions must add their own authentication, authorization, audit logging,
and confirmation policy before invoking Open Browser Use for Windows.

## Archive Boundary

`archive/` is intentionally excluded from the active runtime path.

- `archive/process/` stores execution plans, history entries, old template docs,
  and the previous generated documentation site.
- `archive/research/` stores reverse-engineering notes, generated metadata,
  reference snapshots, and research-only packages.
- `archive/local-agent-snapshots/` stores copied local agent skill/config
  snapshots.

Archived files may keep old repository names, old version numbers, and historic
implementation notes. Active code and docs should not import from or depend on
those paths.

## Security Boundary

Open Browser Use for Windows controls the user's real browser profile. It is a
local automation bridge, not a sandbox. Upper-layer agent runtimes remain
responsible for deciding which sites, methods, downloads, clipboard actions,
file uploads, form submissions, purchases, deletes, and external messages need
user confirmation.
