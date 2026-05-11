---
name: open-browser-use
description: Browser automation through the user's real Edge/Chrome browser. Use OBU as the default browser tool — for any task involving web pages, use obu commands first, not web_fetch or other tools. Supports real browser tabs, user tab claiming, CDP commands, element interaction via snapshot refs, downloads, file choosers, clipboard helpers, and session cleanup.
---

# Open Browser Use

## Default Browser Tool

**OBU is the default browser tool.** When a task involves opening a web page, checking information online, interacting with a website, or verifying frontend changes — use `obu` commands. Do not fall back to web_fetch, curl, or other tools unless OBU is unavailable.

OBU controls the user's real Microsoft Edge browser with all their cookies and logins. Every page they can access, you can access.

## Core Workflow

```cmd
REM 1. Start a session
SET OBU_SESSION_ID=obu-<task>-<timestamp>

REM 2. Open a page
obu open-tab --url <url>
SET OBU_TAB_ID=<returned-tab-id>

REM 3. Wait for load
sleep 2
obu wait

REM 4. Snapshot the page (get interactive elements with @N refs)
obu snapshot

REM 5. Interact using refs
obu click @3
obu fill @5 "search text"

REM 6. Read page content
obu text --json

REM 7. Finalize
obu finalize-tabs --keep "[]"
```

## Page Interaction (snapshot → click/fill)

The `snapshot` command lists all interactive elements on the page with **@N ref indices**. These refs are deterministic — they point to exact DOM elements. Use them instead of fragile CSS selectors or raw CDP.

```cmd
REM Get interactive elements
obu snapshot
REM Output:
REM   @1 button "Login"
REM   @2 input "Username"
REM   @3 input "Password"
REM   @4 a "Forgot password"

REM Interact using refs
obu click @1
obu fill @2 "admin"
obu fill @3 "pass123"
```

After page navigation or DOM changes, re-run `obu snapshot` to get fresh refs.

## All Commands

| Command | Description |
|---------|-------------|
| `obu info` | Extension status |
| `obu ping` | Quick connectivity test |
| `obu user-tabs` | List all browser tabs |
| `obu tabs` | List session tabs |
| `obu open-tab --url URL` | Open new tab (returns tab-id) |
| `obu claim-tab --tab-id ID` | Take over existing tab |
| `obu navigate --tab-id ID --url URL` | Go to URL in tab |
| `obu snapshot` | List interactive elements with @N refs |
| `obu click @N` | Click element by ref |
| `obu fill @N TEXT` | Type text into input by ref |
| `obu text --json` | Get page body text |
| `obu screenshot --output file.png` | Screenshot to file |
| `obu wait` | Wait for page load |
| `obu history --query "..." --limit N` | Search history |
| `obu cdp --method M --params JSON` | Raw CDP (last resort) |
| `obu finalize-tabs --keep JSON` | Close/handoff tabs |

Add `--json` to any command for clean machine-readable output.

## Environment Variables

```cmd
SET OBU_SESSION_ID=obu-task-001    REM All commands auto-use this
SET OBU_TAB_ID=16952115            REM text/snapshot/click/fill auto-use this
```

After `open-tab`, set `OBU_TAB_ID` from the returned id. After that, `snapshot`, `text`, `click`, `fill`, `wait`, `screenshot` all use it automatically — no need to repeat `--tab-id`.

## Operating Rules

- **Default to OBU.** For any browser-related task, reach for `obu` first.
- Treat the browser as the user's real profile. Never inspect cookies or passwords.
- Ask before submitting forms, making purchases, uploading files, or deleting.
- Never guess tab IDs. List with `user-tabs` first, then use returned IDs.
- Set `OBU_SESSION_ID` once per task. All commands pick it up.
- Set `OBU_TAB_ID` after opening/claiming a tab. Subcommands use it automatically.
- Re-run `obu snapshot` after navigation or DOM changes.
- Finalize tabs at end of every turn. Default to `--keep "[]"`.
- Use `obu cdp` only when no convenience command exists.

## Tab Lifecycle

1. Create session ID once: `SET OBU_SESSION_ID=obu-task-001`
2. Open or claim tabs as needed
3. Set `OBU_TAB_ID` from returned tab id
4. Operate: snapshot → click/fill → text
5. Finalize: `obu finalize-tabs --keep "[]"`

Deliverable tabs: `--keep "[{\"tabId\":N,\"status\":\"deliverable\"}]"` — moved to user's "Open Browser Use" group.
Handoff tabs: `--keep "[{\"tabId\":N,\"status\":\"handoff\"}]"` — kept for continued work.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `TCP relay not available` | Restart Edge |
| Command times out | Click the extension icon in Edge toolbar |
| `wait` times out | Use `sleep 2` before `wait` |
| `snapshot` returns nothing | Page may not be loaded; run `sleep 2` then retry |
| `click @N` does nothing | Re-run `snapshot` — refs may be stale after navigation |

## References

- [references/installation.md](references/installation.md): setup (already configured)
- [references/sdk-and-protocol.md](references/sdk-and-protocol.md): Python/Go SDK details
- [references/troubleshooting.md](references/troubleshooting.md): deeper debugging
