---
name: open-browser-use
description: Use when the user mentions `@obu`, `@open-browser-use`, `@real-browser`, asks to use Open Browser Use, or needs a fallback when Codex app Browser cannot start. Browser automation through the user's real Edge/Chrome browser. Supports low-token page-info/text/snapshot extraction, screenshots, element interaction via snapshot refs, downloads, file choosers, clipboard helpers, and session cleanup.
---

# Open Browser Use

OBU controls the user's real Microsoft Edge browser with all their cookies and logins.

## Session Bootstrap

Every browser task starts with this sequence. Don't skip steps.

```cmd
REM 1. Check connectivity (retry once if it fails)
obu ping
REM If setup is uncertain or ping fails twice, run:
obu doctor --browser all --json

REM 2. Start a session
SET OBU_SESSION_ID=obu-<task>-<timestamp>

REM 3. Open or claim a tab
obu open-tab --url <url>
REM Save the returned tab id
SET OBU_TAB_ID=<returned-id>

REM 4. Wait for page load
sleep 2
obu wait
```

After bootstrap, `OBU_TAB_ID` is set — `snapshot`, `text`, `click`, `fill`, `wait`, `screenshot` all use it automatically.

Use `obu page-info --json` when you want a compact page summary with title, URL,
ready state, and body text in one call. Use `obu text --json` when you only
need raw page text.

For long or link-heavy pages, keep extraction bounded:

```cmd
obu text --selector main --max-chars 2000 --json
obu page-info --max-chars 2000 --json
obu snapshot --limit 50 --json
```

For auditable multi-step browser work, add a trace log to an action plan:

```cmd
obu run --trace-log trace.jsonl -c "ping"
```

Each JSONL row records the session, turn, action, risk class, tab id, duration,
and success/error status. `obu mcp --trace-log trace.jsonl` records the same
trace format for direct MCP tool calls.

For multimodal models, use screenshots as visual keyframes rather than the main
data channel: bounded `page-info` / `text` for content, `snapshot` only when
refs are needed, and one scoped or viewport `screenshot` for visual state.
Screenshot results return local file metadata; do not inline image data into
stdout or trace logs.

`obu run` writes partial JSON output even when an action fails. Use
`--continue-on-error` when later diagnostic steps should still run after a
failure, for example to collect text or a final visual keyframe.

## Page Interaction

### Snapshot → click/fill (preferred)

When you need to interact with page elements — click buttons, fill forms, navigate menus — use the snapshot workflow. It's faster and more reliable than raw CDP.

```cmd
REM Get the page's interactive elements
obu snapshot
REM Output:
REM   @1 button "Login"
REM   @2 input "Username"
REM   @3 input "Password"
REM   @4 a "Sign up"

REM Interact using refs
obu click @1 --json
obu fill @2 "admin" --json
obu fill @3 "pass123" --json
```

`click --json` and `fill --json` return `{ "ok": true, "action": ..., "ref": ... }`
on success. On failure they include a `reason` such as `not-found`,
`disabled`, `not-visible`, or `not-fillable`; take a fresh snapshot before
retrying.

### Read text

When you just need to extract page content — no interaction needed:

```cmd
obu text --max-chars 2000 --json
```

### Read page summary

When you want a small machine-readable page summary instead of only the body
text:

```cmd
obu page-info --max-chars 2000 --json
```

### Decision: snapshot vs text vs cdp

| Situation | Use |
|-----------|-----|
| Need to click, fill, or find elements | `obu snapshot` then `click @N` / `fill @N` |
| Need title, URL, ready state, and text together | `obu page-info --json` |
| Just need page text content | `obu text --json` |
| Need specific data (title, URL, attribute) | `obu cdp --method Runtime.evaluate ...` |
| Multi-step workflow | `obu run` action plan |
| Screenshot for visual check | `obu screenshot --output file.png --json` |

### Snapshot Discipline

- **Take a fresh snapshot after**: navigation, page reload, opening/closing modals, form submission, or any DOM change.
- **Reuse the snapshot**: if the page hasn't changed since your last snapshot, the refs are still valid.
- **Stale refs**: if `click @N` does nothing or targets wrong element, re-run `obu snapshot`.
- **100 element limit**: if the target isn't in the first 100 elements, scope with navigation or use CDP.

## All Commands

| Command | Use |
|---------|-----|
| `obu doctor --browser all --json` | Diagnose Chrome and Edge native host, manifest, relay, and extension status |
| `obu doctor --json` | Diagnose one browser integration, defaulting to Chrome |
| `obu ping` | Quick connectivity check |
| `obu info` | Extension status and version |
| `obu user-tabs` | List all browser tabs |
| `obu open-tab --url URL` | Open new tab → returns tab id |
| `obu claim-tab --tab-id ID` | Take over existing user tab |
| `obu navigate --tab-id ID --url URL` | Go to URL in tab |
| `obu page-info --json` | Get title, URL, ready state, and page body text |
| `obu text --selector CSS --max-chars N --json` | Get bounded page or selector text |
| `obu snapshot --limit N --json` | Get a bounded interactive element list |
| `obu snapshot` | List interactive elements with @N refs |
| `obu click @N --json` | Click element by ref and return action status |
| `obu fill @N TEXT --json` | Fill element by ref and return action status |
| `obu text --json` | Get page body text |
| `obu screenshot --output file.png --json` | Screenshot current viewport |
| `obu screenshot --selector CSS --output file.png --json` | Screenshot one element |
| `obu screenshot --full-page --output file.png --json` | Screenshot the full page |
| `obu wait` | Wait for page readyState complete |
| `obu history --query "..."` | Search browser history |
| `obu cdp --method M --params JSON` | Raw CDP (last resort) |
| `obu finalize-tabs --keep JSON` | Close or handoff session tabs |
| `obu run -c "..."` | Multi-step action plan |

All commands accept `--json` for machine-readable output and respect `OBU_SESSION_ID` / `OBU_TAB_ID` env vars.

For the common read paths, prefer these stable `--json` shapes:

- `obu page-info --json` -> title, URL, ready state, and body text
- `obu text --json` -> `{ "text": ... }`
- `obu snapshot --json` -> `{ "items": [...] }`
- `obu screenshot --json` -> `{ "path": ..., "bytes": ..., "format": "png", "clip": ...? }`
- `obu history --json` -> `{ "items": [...] }`
- `obu wait --json` -> `{ "readyState": ... }`

For common action paths, prefer these stable `--json` shapes:

- `obu doctor --json` -> `{ "ok": ..., "socket": ..., "nativeHost": ..., "browserExtension": ..., "checks": [...] }`
- `obu doctor --browser all --json` -> `{ "ok": ..., "browsers": [...], "nextSteps": [...] }`
- `obu ping --json` -> `{ "status": "pong" }`
- `obu claim-tab --json` -> `{ "tab": ... }`
- `obu navigate --json` -> `{ "navigate": ... }`
- `obu name-session --json` -> `{ "ok": true }`
- `obu finalize-tabs --json` -> `{ "ok": true }`
- `obu turn-ended --json` -> `{ "ok": true }`

## Risk Labels

OBU labels traced actions so the surrounding agent runtime can decide whether
to ask for confirmation:

| Risk | Actions |
|------|---------|
| `read` | `doctor`, `ping`, `info`, `tabs`, `user-tabs`, `history`, `wait`, `page-info`, `text`, `snapshot`, `screenshot` |
| `navigation` | `open-tab`, `claim-tab`, `navigate` |
| `interaction` | `click`, `fill`, `move-mouse` |
| `file-system` | `set-file-chooser-files` |
| `session` | `name-session`, `turn-ended`, `finalize-tabs` |
| `unrestricted` | `cdp`, `call` |

Treat `interaction`, `file-system`, and `unrestricted` actions as sensitive
when they could submit, upload, delete, purchase, send, or expose private data.
The CLI records the label but does not enforce confirmation for you.

## Error Recovery

| Symptom | Action |
|---------|--------|
| `ping` fails | Wait 2s, retry once. Still failing → run `obu doctor --browser all --json`, then restart Edge if the relay is unavailable. |
| `open-tab` returns no id | Host may have disconnected. Wait 2s, retry. |
| `wait` times out | Page load is slow. Use `sleep 3` then try `text` directly. |
| `snapshot` returns nothing | Page not loaded yet. `sleep 2` then retry. |
| `click @N` does nothing | Re-run `obu snapshot` — refs are stale. |
| `text` returns empty | Page may be a SPA. Try `sleep 2` then retry. |
| `TCP relay not available` | Host process not running. Restart Edge. |
| Command hangs | Extension may be idle. Click the OBU extension icon in Edge toolbar. |

## Session Cleanup

At the end of every browser turn, finalize tabs:

```cmd
REM Close all session tabs (default)
obu finalize-tabs --keep "[]"

REM Keep a tab for the user to inspect
obu finalize-tabs --keep "[{\"tabId\":N,\"status\":\"deliverable\"}]"

REM Keep a tab for continued work
obu finalize-tabs --keep "[{\"tabId\":N,\"status\":\"handoff\"}]"
```

- Do not call any OBU commands after finalizing.
- Omit research, intermediate, and source tabs by default.
- Keep only tabs the user explicitly needs.

## Operating Rules

- Treat the browser as the user's real profile. Never inspect cookies, passwords, or session data.
- Ask before submitting forms, making purchases, uploading files, or deleting.
- Never guess tab IDs — always list with `user-tabs` first.
- One session per task. Don't reuse `obu-cli`.
- Run `obu ping` at the start of every browser task. Retry once on failure, then use `obu doctor --browser all --json` for diagnostics.
- After `open-tab` or `claim-tab`, set `OBU_TAB_ID` from the returned id.
- Re-run `obu snapshot` after any page navigation or DOM state change.
- Finalize tabs at the end of every turn.

## References

- [references/installation.md](references/installation.md): setup guide (already configured)
- [references/sdk-and-protocol.md](references/sdk-and-protocol.md): Python/Go SDK details
- [references/troubleshooting.md](references/troubleshooting.md): deeper debugging
