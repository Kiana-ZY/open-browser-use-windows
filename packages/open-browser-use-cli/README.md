# Open Browser Use CLI

This npm package installs the `open-browser-use` native host and CLI binary.
It also exposes the short `obu` command.

The CLI is useful when your agent runtime is shell-first or when you want a
small action plan instead of a JavaScript/Python SDK script.

## Install

```sh
npm install -g open-browser-use
obu version
```

The package contains prebuilt Go binaries for macOS, Linux, and Windows on
`amd64` and `arm64`.

## Setup

Run `open-browser-use` with no subcommand to print the CLI version, browser
extension status, extension version when available, and the next setup or
upgrade command.

After installation, run setup to register the Chromium native messaging host
and guide Chrome or Edge extension installation:

```sh
open-browser-use setup
open-browser-use setup --browser edge
```

While the Chrome Web Store item is pending review, use the latest GitHub Release
zip as an unpacked extension instead:

```sh
open-browser-use setup beta
```

That command opens `chrome://extensions/` or `edge://extensions/` and reveals
the keyed release ZIP so the user can drag it into the browser with the
expected extension id.

Verify the browser connection:

```sh
open-browser-use doctor
open-browser-use doctor --json
open-browser-use doctor --browser all --json
open-browser-use ping
open-browser-use info
```

Use `doctor` first when setup fails or `ping` cannot connect. It checks the
CLI version, OS, selected browser, stable native host path, native messaging
manifest, relay connectivity, and extension version. On Windows it also checks
the native messaging registry key for Chrome or Edge. Use `--browser all` when
an agent or issue template needs one preflight covering both browser routes.

## One-Shot Commands

Use direct subcommands for simple inspection and single browser actions:

```sh
open-browser-use tabs
open-browser-use user-tabs
open-browser-use history --query "browser use" --limit 20

open-browser-use name-session --name "Docs scan - OBU"
open-browser-use open-tab --url https://docs.browser-use.com
open-browser-use navigate --tab-id <tab-id> --url https://github.com/Kiana-ZY/open-browser-use-windows
open-browser-use page-info --tab-id <tab-id>
open-browser-use text --tab-id <tab-id>
open-browser-use cdp --tab-id <tab-id> --method Runtime.evaluate --params '{"expression":"document.title","returnByValue":true}'
open-browser-use finalize-tabs --keep '[]'
```

Use `--socket <path>` when a runtime gives you a specific Open Browser Use
relay path. On Windows, the default relay is `127.0.0.1:19832`; on Unix-like
systems the CLI discovers `/tmp/open-browser-use/active.json`.
Direct subcommands and `run` use the same default browser session,
`obu-cli`, so a final `open-browser-use finalize-tabs --keep '[]'` cleans up
tabs opened by either style. Pass `--session-id <id>` only when you intentionally
want a separate tab group and cleanup scope.

Use `page-info` when you want one compact response containing title, URL,
`readyState`, and body text. Use `text` when you only need raw page text.
For long pages, bound the payload with `--max-chars`; for scoped extraction,
pass `--selector`. For link-heavy pages, reduce or expand snapshot output with
`--limit`:

```sh
open-browser-use text --tab-id <tab-id> --selector main --max-chars 2000 --json
open-browser-use page-info --tab-id <tab-id> --max-chars 2000 --json
open-browser-use snapshot --tab-id <tab-id> --limit 50 --json
open-browser-use screenshot --tab-id <tab-id> --selector main --output main.png --json
```

For GPT-5-class multimodal agents, treat screenshots as visual keyframes rather
than the main data channel. Use bounded `page-info` / `text` for content,
`snapshot` only when interaction refs are needed, and one viewport or scoped
`screenshot` to verify visual state. Screenshots return local file metadata and
never inline base64 in stdout.

For the high-frequency read commands, `--json` now returns stable object shapes:

- `open-tab --json` -> `{ "tab": ..., "navigate": ...? }`
- `page-info --json` -> `{ "title": ..., "url": ..., "readyState": ..., "text": ... }`
- `text --json` -> `{ "text": ... }`
- `snapshot --json` -> `{ "items": [...] }`
- `screenshot --json` -> `{ "path": ..., "bytes": ..., "format": "png", "clip": ...? }`
- `history --json` -> `{ "items": [...] }`
- `wait --json` -> `{ "readyState": ... }`

For common action commands, `--json` also prefers stable object shapes:

- `doctor --json` -> `{ "ok": ..., "version": ..., "socket": ..., "nativeHost": ..., "browserExtension": ..., "checks": [...] }`
- `doctor --browser all --json` -> `{ "ok": ..., "version": ..., "browsers": [...], "nextSteps": [...] }`
- `ping --json` -> `{ "status": "pong" }`
- `claim-tab --json` -> `{ "tab": ... }`
- `navigate --json` -> `{ "navigate": ... }`
- `name-session --json` -> `{ "ok": true }`
- `finalize-tabs --json` -> `{ "ok": true }`
- `turn-ended --json` -> `{ "ok": true }`

## Action Plans

Use `run` when you want CLI-level orchestration without writing JS or Python.
The format is intentionally small: one action per line, optional `#` comments,
shell-like quotes, shared session/turn, and a default tab set by `open-tab` or
`claim-tab`.

```sh
open-browser-use run -c '
name-session "Docs scan - OBU"
open-tab https://docs.browser-use.com
wait-load domcontentloaded
page-info --max-chars 2000
text --max-chars 2000
snapshot --limit 50
screenshot --selector main --output main.png
click @1
fill @2 "hello"
finalize-tabs []
'
```

You can also load the same action plan from a file:

```sh
open-browser-use run --file ./docs-scan.obu
```

Add `--trace-log <path>` when you want a local JSONL audit trail for an action
plan. Each line records `timestamp`, `sessionId`, `turnId`, action line,
action name, risk class, tab id, duration, success flag, and error text when
the action fails.

If an action fails, `run` still writes a partial JSON result before returning a
non-zero exit code. The top-level output includes `ok: false`, `error`, and the
failed step with its own `ok: false` and `error`. Use `--continue-on-error`
when an agent should keep collecting later diagnostic steps such as text or a
visual keyframe after an earlier action fails.

Supported actions:

- Session/info: `ping`, `info`, `tabs`, `user-tabs`, `turn-ended`
- Browser tabs: `open-tab`, `claim-tab`, `navigate`, `wait-load`, `page-info`, `text`, `snapshot`, `screenshot`, `click`, `fill`
- Browser methods: `history`, `cdp`, `call`
- Input/files: `move-mouse`, `wait-file-chooser`, `set-file-chooser-files`
- Cleanup: `finalize-tabs`

Example with explicit CDP and default tab reuse:

```sh
open-browser-use run -c '
name-session "GitHub issue scan - OBU"
open-tab https://github.com/Kiana-ZY/open-browser-use-windows/issues
wait-load domcontentloaded
cdp Runtime.evaluate "{\"expression\":\"document.body.innerText.slice(0, 1000)\",\"returnByValue\":true}"
finalize-tabs []
'
```

Example claiming an existing user tab:

```sh
open-browser-use run -c '
claim-tab <tab-id>
page-info
finalize-tabs [{"tabId":<tab-id>,"status":"handoff"}]
'
```

## MCP Server

Use `mcp` when an agent runtime supports local MCP servers over stdio:

```sh
obu mcp
```

For Codex, add a server entry similar to this in `~/.codex/config.toml`:

```toml
[mcp_servers.open_browser_use]
command = "obu"
args = ["mcp"]
```

The MCP server exposes browser tools such as `doctor`, `user_tabs`, `open_tab`,
`claim_tab`, `navigate`, `wait_load`, `page_info`, `text`, `snapshot`,
`screenshot`, `click`, `fill`, `cdp`, `history`, `run_action_plan`,
`finalize_tabs`, and unrestricted `call`. It uses the same
relay discovery as the CLI; pass `--socket <path>` or `--socket-dir <dir>` in
`args` only when the runtime needs an explicit relay.

Pass `--trace-log <path>` in the MCP `args` to write the same JSONL trace for
direct MCP tool calls. `run_action_plan` writes per-action trace lines through
the runner, so the MCP wrapper does not add a duplicate plan-level row.

For the common high-frequency tools, the MCP server mirrors the same stable
object shapes documented for CLI `--json` via `structuredContent`, so agents do
not need separate adapters for CLI vs MCP on `doctor`, `ping`, `tabs`,
`user_tabs`, `history`, `open_tab`, `claim_tab`, `navigate`, `page_info`,
`text`, `snapshot`, `screenshot`, `click`, `fill`, `wait_load`,
`name_session`, `finalize_tabs`, and `turn_ended`.

## Safety And Observability

Every browser JSON-RPC request receives a `request_id` when the caller has not
already supplied one. Action plans and traced MCP tool calls also classify each
step:

- `read`: `doctor`, `ping`, `info`, `tabs`, `user-tabs`, `history`, `wait-load`, `page-info`, `text`, `snapshot`, `screenshot`
- `navigation`: `open-tab`, `claim-tab`, `navigate`
- `interaction`: `click`, `fill`, `move-mouse`
- `file-system`: `set-file-chooser-files`
- `session`: `name-session`, `turn-ended`, `finalize-tabs`
- `unrestricted`: `cdp`, `call`

Open Browser Use records these labels for observability. It does not enforce a
Codex-style confirmation policy itself; integrations should ask the user before
uploads, downloads, clipboard access, form submission, purchases, deletion,
external message sending, or any unrestricted `cdp` / `call` that can create
side effects.

## Low-Level JSON-RPC

Use `call` when no convenience command or action exists:

```sh
open-browser-use call --method getInfo --params '{}'
open-browser-use call --method executeCdp --params '{"target":{"tabId":123},"method":"Runtime.evaluate","commandParams":{"expression":"document.title","returnByValue":true}}'
```

## Cleanup

End browser work by finalizing tabs:

```sh
open-browser-use finalize-tabs --keep '[]'
```

Keep a tab only when the live tab is the deliverable or the user should continue
from that state:

```sh
open-browser-use finalize-tabs --keep '[{"tabId":123,"status":"handoff"}]'
```
