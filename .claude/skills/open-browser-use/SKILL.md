---
name: open-browser-use
description: Browser automation through the user's real Edge/Chrome browser. Supports real browser tabs, user tab claiming, CDP commands, element interaction via snapshot refs, downloads, file choosers, clipboard helpers, and session cleanup.
---

# Open Browser Use

OBU controls the user's real Microsoft Edge browser with all their cookies and logins.

## Session Bootstrap

Every browser task starts with this sequence. Don't skip steps.

```cmd
REM 1. Check connectivity (retry once if it fails)
obu ping
REM If ping fails: wait 2s, then obu ping again. Still failing? See Troubleshooting.

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
obu click @1
obu fill @2 "admin"
obu fill @3 "pass123"
```

### Read text

When you just need to extract page content — no interaction needed:

```cmd
obu text --json
```

### Decision: snapshot vs text vs cdp

| Situation | Use |
|-----------|-----|
| Need to click, fill, or find elements | `obu snapshot` then `click @N` / `fill @N` |
| Just need page text content | `obu text --json` |
| Need specific data (title, URL, attribute) | `obu cdp --method Runtime.evaluate ...` |
| Multi-step workflow | `obu run` action plan |
| Screenshot for visual check | `obu screenshot --output file.png` |

### Snapshot Discipline

- **Take a fresh snapshot after**: navigation, page reload, opening/closing modals, form submission, or any DOM change.
- **Reuse the snapshot**: if the page hasn't changed since your last snapshot, the refs are still valid.
- **Stale refs**: if `click @N` does nothing or targets wrong element, re-run `obu snapshot`.
- **100 element limit**: if the target isn't in the first 100 elements, scope with navigation or use CDP.

## All Commands

| Command | Use |
|---------|-----|
| `obu ping` | Quick connectivity check |
| `obu info` | Extension status and version |
| `obu user-tabs` | List all browser tabs |
| `obu open-tab --url URL` | Open new tab → returns tab id |
| `obu claim-tab --tab-id ID` | Take over existing user tab |
| `obu navigate --tab-id ID --url URL` | Go to URL in tab |
| `obu snapshot` | List interactive elements with @N refs |
| `obu click @N` | Click element by ref |
| `obu fill @N TEXT` | Type into input by ref |
| `obu text --json` | Get page body text |
| `obu screenshot --output file.png` | Screenshot to file |
| `obu wait` | Wait for page readyState complete |
| `obu history --query "..."` | Search browser history |
| `obu cdp --method M --params JSON` | Raw CDP (last resort) |
| `obu finalize-tabs --keep JSON` | Close or handoff session tabs |
| `obu run -c "..."` | Multi-step action plan |

All commands accept `--json` for machine-readable output and respect `OBU_SESSION_ID` / `OBU_TAB_ID` env vars.

## Error Recovery

| Symptom | Action |
|---------|--------|
| `ping` fails | Wait 2s, retry once. Still failing → restart Edge. |
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
- Run `obu ping` at the start of every browser task. Retry once on failure.
- After `open-tab` or `claim-tab`, set `OBU_TAB_ID` from the returned id.
- Re-run `obu snapshot` after any page navigation or DOM state change.
- Finalize tabs at the end of every turn.

## References

- [references/installation.md](references/installation.md): setup guide (already configured)
- [references/sdk-and-protocol.md](references/sdk-and-protocol.md): Python/Go SDK details
- [references/troubleshooting.md](references/troubleshooting.md): deeper debugging
