# Open Browser Use for Windows

Open Browser Use for Windows is a Windows-first build of Open Browser Use. It
connects a Chromium extension, a local Go native host, and CLI/SDK/MCP clients
so agents can control the user's real Chrome or Microsoft Edge profile.

The command name stays `open-browser-use`, with `obu` as the short alias.

## What This Fork Focuses On

| Area | Windows support |
| --- | --- |
| Browser support | Chrome and Microsoft Edge |
| Native host | `open-browser-use.exe` launched by Chrome/Edge Native Messaging |
| Client transport | TCP relay on `127.0.0.1:19832` for Windows |
| Agent access | CLI, MCP server, JavaScript SDK, Python SDK, Go SDK, and skill docs |
| Setup helpers | `setup`, `setup beta`, and `install-manifest` support `--browser edge` |

## Architecture

```text
Chrome / Edge MV3 extension
  -> Native Messaging stdio
  -> open-browser-use.exe
  -> TCP 127.0.0.1:19832 on Windows
  -> CLI / MCP / JS SDK / Python SDK / Go SDK
```

## Build

```powershell
go build -o open-browser-use.exe ./cmd/open-browser-use
```

## Install The Browser Integration

For Edge:

```powershell
.\open-browser-use.exe setup beta --browser edge
```

For Chrome:

```powershell
.\open-browser-use.exe setup beta --browser chrome
```

The setup command opens the browser extension page and reveals the extension ZIP.
Enable Developer mode, then drag the ZIP into the extensions page.

## Verify

```powershell
.\open-browser-use.exe version
.\open-browser-use.exe ping --json
.\open-browser-use.exe info --json
```

When installed globally, the same checks are:

```powershell
obu version
obu ping --json
obu info --json
```

## Common Commands

Use one unique session id per task.

```powershell
$env:OBU_SESSION_ID = "obu-task-$(Get-Date -Format yyyyMMddHHmmss)"
obu name-session --name "Task - OBU"
obu open-tab --url https://github.com/trending --json
obu page-info --max-chars 2000 --json
obu snapshot --limit 50 --json
obu finalize-tabs --keep "[]" --json
```

Useful commands:

| Command | Purpose |
| --- | --- |
| `obu ping --json` | Connectivity check |
| `obu info --json` | Extension and native host metadata |
| `obu user-tabs --json` | List all browser tabs |
| `obu open-tab --url URL --json` | Open a managed task tab |
| `obu claim-tab --tab-id ID --json` | Claim an existing user tab |
| `obu page-info --max-chars N --json` | Compact title, URL, ready state, and text |
| `obu snapshot --limit N --json` | Bounded interactive element list |
| `obu click @1 --json` | Click a snapshot ref |
| `obu fill @2 "text" --json` | Fill a snapshot ref |
| `obu screenshot --output file.png --json` | Capture a screenshot |
| `obu run -c "..."` | Run a small action plan |
| `obu mcp` | Start the stdio MCP server |

## SDKs

JavaScript/TypeScript:

```bash
npm install open-browser-use-sdk
```

Python:

```bash
pip install open-browser-use-sdk
```

Go:

```bash
go get github.com/Kiana-ZY/open-browser-use-windows/packages/open-browser-use-go
```

## Agent Skill

The reusable agent skill lives in `skills/open-browser-use/`. It is designed for
Codex, Claude Code, and other shell-first agent runtimes. The trigger alias is
`@obu`; the CLI alias remains `obu`.

See `docs/CODEX_AND_CLAUDE_USAGE.md` for Codex and Claude Code MCP examples.

## Repository Layout

| Path | Purpose |
| --- | --- |
| `apps/chrome-extension/` | MV3 extension for Chrome and Edge |
| `cmd/open-browser-use/` | Go CLI, native host, and MCP server |
| `internal/host/` | Native host relay and Windows TCP transport |
| `internal/wire/` | Native messaging frame codec |
| `packages/open-browser-use-cli/` | npm CLI package with `open-browser-use` and `obu` |
| `packages/open-browser-use-js/` | JavaScript/TypeScript SDK |
| `packages/open-browser-use-python/` | Python SDK |
| `packages/open-browser-use-go/` | Go SDK |
| `skills/open-browser-use/` | Agent-facing usage instructions |
| `archive/` | Non-core research, process history, and local snapshots |

## Archive

The `archive/` directory keeps material that is useful for provenance but not
needed for the Windows runtime itself:

- `archive/process/` stores execution plans, histories, old template docs, and
  the previous standalone HTML guide.
- `archive/research/` stores reverse-engineering notes, generated metadata, and
  research-only packages.
- `archive/local-agent-snapshots/` stores copied local agent skill snapshots.

## License

MIT. This project is derived from the upstream Open Browser Use work and adapted
for a Windows-first Chrome/Edge workflow.
