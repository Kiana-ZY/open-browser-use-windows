---
name: open-browser-use
description: Platform-neutral guidance for using Open Browser Use, the open-source browser automation stack for AI agents. Supports Chrome and Edge. Use when an agent needs to install, verify, troubleshoot, or operate Open Browser Use through its browser extension, native CLI, JavaScript SDK, Python SDK, Go SDK, or Browser Use style JSON-RPC methods; use for tasks involving real browser tabs, user tab claiming, CDP commands, downloads, file choosers, clipboard helpers, or session cleanup.
---

# Open Browser Use

## Overview

Open Browser Use connects an MV3 browser extension, a local native messaging host, a CLI, SDKs, and an optional stdio MCP server so agents can automate a real browser profile (Chrome or Edge). It is not Codex.app-specific; adapt the commands, MCP config, and SDK examples to the agent runtime you are operating in.

## Supported Browsers

Open Browser Use supports Chrome and Edge. On Windows, pass `--browser edge` to `setup`, `setup beta`, and `install-manifest` commands to configure Edge instead of Chrome. The browser extension is the same MV3 extension; only the native messaging host registration path differs. Once configured, all other CLI commands work identically regardless of browser.

## Core Workflow

1. Check setup with `obu ping`. If it fails because setup is missing, read [references/installation.md](references/installation.md).
2. Set `OBU_SESSION_ID` env var for the current agent task. Use a unique id such as `obu-<task-slug>-<timestamp>`. All subsequent commands pick it up automatically.
3. Name the current browser task group before opening or claiming tabs. Use a short task label followed by ` - OBU`; if no better task label is available, use `Task - OBU`.
4. Use `obu user-tabs` to list tabs, `obu open-tab --url <url>` to open new ones, `obu claim-tab --tab-id <id>` to take over existing ones.
5. After opening/claiming a tab, set `OBU_TAB_ID=<id>` env var. Then use `obu text --json`, `obu screenshot --output <path>`, `obu wait`, or `obu cdp` without repeating `--tab-id`.
6. Use `obu run` for multi-step action plans, or the Python/Go SDK for event-driven workflows. Read [references/sdk-and-protocol.md](references/sdk-and-protocol.md).
7. Before ending browser work, finalize tabs with `obu finalize-tabs --keep "<json-array>"`.
8. If communication fails, read [references/troubleshooting.md](references/troubleshooting.md).

## Operating Rules

- Treat the browser as the user's real browser profile. Do not inspect cookies, passwords, session stores, or unrelated browser data.
- Ask the user before installing the extension, opening the browser for them, enabling extension permissions, uploading local files, reading/writing clipboard data, submitting forms, purchasing, deleting, sending, or making other externally visible changes.
- Do not assume Codex.app helpers, Node REPL globals, or a bundled plugin UI are available. Use the installed `open-browser-use` / `obu` CLI or the published SDKs.
- Do not guess tab ids. List tabs first, then use ids returned by `tabs`, `user-tabs`, `open-tab`, or SDK calls.
- Prefer `claim-tab` / `claimUserTab` for existing user tabs. Claiming should be based on the current `user-tabs` result and visible evidence such as URL, title, recency, or group.
- Use `--socket` only when the user or runtime provides an explicit socket. Otherwise let the CLI and SDKs discover the active socket registry.
- Set `OBU_SESSION_ID` env var at the start of a task. All commands pick it up automatically — no need to repeat `--session-id` on every call.
- Set `OBU_TAB_ID` env var after opening or claiming a tab. Subsequent `text`, `screenshot`, `wait`, `cdp` commands use it automatically.
- Direct CLI subcommands and `open-browser-use run` can share the same browser session only when they use the same explicit `--session-id`. Finalize that same session before ending browser work.
- Use `call --method <method> --params "<json>"` only when no safer convenience command or SDK wrapper exists.
- Add `--json` to any command for machine-readable output (result value only, no JSON-RPC wrapper). Useful for piping and scripting.

## Common CLI Actions

Set environment variables once per session, then commands are terse:

On Windows (CMD):

```cmd
SET OBU_SESSION_ID=obu-task-20260511
obu ping
obu info
obu name-session --name "Task - OBU"
obu tabs
obu user-tabs
obu history --query "example" --limit 20
obu open-tab --url https://example.com
SET OBU_TAB_ID=<returned-tab-id>
obu text --json
obu screenshot --output page.png
obu wait --state load
obu cdp --method Runtime.evaluate --params "{\"expression\":\"document.title\"}"
obu finalize-tabs --keep "[]"
```

On Unix (macOS/Linux):

```sh
export OBU_SESSION_ID="obu-task-$(date +%Y%m%d%H%M%S)"
obu ping
obu info
obu name-session --name "Task - OBU"
obu tabs
obu user-tabs
obu history --query "example" --limit 20
obu open-tab --url https://example.com
export OBU_TAB_ID=<returned-tab-id>
obu text --json
obu screenshot --output page.png
obu wait --state load
obu cdp --method Runtime.evaluate --params '{"expression":"document.title"}'
obu finalize-tabs --keep '[]'
```

### Convenience Commands

| Command | Description |
|---------|-------------|
| `obu text --json` | Get page body text (respects OBU_TAB_ID) |
| `obu screenshot --output file.png` | Screenshot to file |
| `obu wait --state load` | Wait for page readyState |

For CLI-level orchestration without writing SDK code, use a line-oriented action plan:

```cmd
REM Windows
obu run --session-id %OBU_SESSION_ID% -c "name-session Task - OBU
open-tab --url https://example.com
wait-load domcontentloaded
page-info
finalize-tabs []"
```

```sh
# Unix
obu run --session-id "$OBU_SESSION_ID" -c '
name-session "Task - OBU"
open-tab https://example.com
wait-load domcontentloaded
page-info
finalize-tabs []
'
```

Each action line shares one session/turn. `open-tab` and `claim-tab` set the default tab for later tab-scoped actions such as `wait-load`, `page-info`, `navigate`, `cdp`, `move-mouse`, and `wait-file-chooser`.

Use `obu` as the short alias when available.

## Tab Lifecycle

- Session tabs are tabs Open Browser Use has created or claimed for the current agent workflow.
- Use one unique session id per agent task or conversation. Do not share the fallback `obu-cli` session across unrelated tasks.
- Task session groups should be named from the task, using the pattern `<short task> - OBU`. Use `Task - OBU` as the fallback name.
- Keep no tabs by default: `obu finalize-tabs --session-id <id> --keep "[]"`.
- Keep a tab only when the user needs that live page after the turn. Omit research, source, search, intermediate, duplicate, blank, error, and login/navigation tabs after extracting what you need.
- Keep a tab with `status: "deliverable"` when the tab itself is the user-facing output or requested open page, such as a created or edited document, dashboard, checkout/cart, submitted form result, or a page the user explicitly asked to inspect directly.
- Keep a tab with `status: "handoff"` only when the task is still in progress and the user or a later turn should continue from the current task group, such as a page waiting for user input, login, approval, payment, CAPTCHA, or an unfinished workflow.
- Handoff tabs stay in the task session group. Deliverable tabs move to the shared `✅ Open Browser Use` tab group.
- Run finalization as the last Open Browser Use browser action for the turn. Do not call Open Browser Use browser tools after finalizing; if more browser work is needed, do it first and finalize once with the final tab disposition.

## File Choosers, Downloads, And Clipboard

- File uploads use the intercepted file chooser flow: start waiting, trigger the chooser in the page, then set absolute local paths with `set-file-chooser-files` or the SDK equivalent.
- Downloads can be observed with SDK notification handlers or Browser Use methods such as `waitForDownload` and `downloadPath`.
- Clipboard helpers operate through the current controlled tab and should be treated as sensitive user actions.

## References

- [references/installation.md](references/installation.md): one-time CLI and browser extension setup, including cases where user cooperation is required.
- [references/sdk-and-protocol.md](references/sdk-and-protocol.md): JavaScript, Python, Go, socket, and JSON-RPC usage details.
- [references/troubleshooting.md](references/troubleshooting.md): connection failures, stale sockets, extension/native host checks, and permission issues.
