# Open Browser Use SDK And Protocol

Read this reference when the task requires multi-step automation, integration into another agent runtime, or direct Browser Use style JSON-RPC calls.

## Connection Model

The Chrome/Edge extension starts the native host through Chromium Native
Messaging. On Windows the native host exposes a localhost TCP relay at
`127.0.0.1:19832`; on Unix-like systems it exposes a local socket and writes the
active socket registry so the CLI and SDKs can discover it.

Default route:

```text
agent runtime
  -> open-browser-use CLI, MCP server, or SDK
  -> active Open Browser Use relay
  -> native messaging host
  -> Chrome/Edge extension
  -> browser tabs / debugger / history / downloads
```

Pass an explicit relay path only when the runtime provides one:

```sh
open-browser-use ping --socket 127.0.0.1:19832
```

For SDKs, create a client with `socketPath` / `socket_path` / `SocketPath`.
Use `127.0.0.1:19832` on Windows, or the active socket path on Unix-like
systems.

## Browser Session Scope

Use a unique browser session id for each agent task or conversation. Prefer a
stable session/conversation id from the surrounding runtime when it exists;
otherwise create a short unique id such as `obu-<task-slug>-<timestamp>`.

Pass that same id through every CLI command, MCP server, or SDK client used for
the task. Do not rely on the CLI fallback `obu-cli` in agent workflows; it is a
manual convenience fallback and can reuse stale Chrome tab groups from unrelated
tasks.

Install the SDK package from the package registry for your runtime:

```sh
npm install open-browser-use-sdk
pip install open-browser-use-sdk
go get github.com/Kiana-ZY/open-browser-use-windows/packages/open-browser-use-go
```

The Python distribution is named `open-browser-use-sdk`, while the import module
is `open_browser_use`. Go code usually imports the package as `obu`.

## JavaScript SDK Pattern

Use the high-level browser helper for common multi-step flows:

```ts
import { connectOpenBrowserUse } from "open-browser-use-sdk";

const browser = await connectOpenBrowserUse({
  socketPath: "127.0.0.1:19832",
  sessionId: "obu-docs-scan-20260510",
});

try {
  await browser.client.nameSession("Task - OBU");
  const tab = await browser.newTab();
  await tab.goto("https://example.com", { waitUntil: "domcontentloaded" });
  const info = await tab.pageInfo({ maxChars: 2000 });
  const snapshot = await tab.snapshot({ limit: 50 });
  console.log(info.title, snapshot.items.length);
} finally {
  await browser.client.finalizeTabs([]);
  browser.close();
}
```

Use the low-level client when you need direct Browser Use JSON-RPC/CDP calls:

```ts
import { OpenBrowserUseClient } from "open-browser-use-sdk";

const client = new OpenBrowserUseClient({
  socketPath: "127.0.0.1:19832",
  sessionId: "obu-docs-scan-20260510",
});

await client.connect();
await client.nameSession("Task - OBU");
const tab = await client.createTab() as { id: number };
await client.executeCdp(tab.id, "Page.navigate", { url: "https://example.com" });
await client.finalizeTabs([]);
client.close();
```

The JavaScript SDK supports notification handlers:

```ts
const unsubscribe = client.onNotification((event) => {
  if (event.method === "onDownloadChange") {
    console.log(event.params);
  }
});
```

## Python SDK Pattern

```py
from open_browser_use import connect_open_browser_use

browser = connect_open_browser_use(
    socket_path="127.0.0.1:19832",
    session_id="obu-issue-scan-20260510",
)

try:
    browser.client.name_session("Issue scan - OBU")
    tab = browser.new_tab()
    tab.goto("https://github.com/Kiana-ZY/open-browser-use-windows/issues", wait_until="domcontentloaded")
    tab.playwright.wait_for_load_state(state="domcontentloaded", timeout=15)
    tab.playwright.wait_for_timeout(1500)

    result = tab.page_info(max_chars=4000)
    result["snapshot"] = tab.snapshot(limit=50)
    print(result)
finally:
    browser.client.finalize_tabs([])
    browser.close()
```

Use the low-level client when you need raw JSON-RPC/CDP calls:

```py
from open_browser_use import OpenBrowserUseClient

client = OpenBrowserUseClient(
    socket_path="127.0.0.1:19832",
    session_id="obu-docs-scan-20260510",
)

client.name_session("Task - OBU")
tab = client.create_tab()
client.execute_cdp(tab["id"], "Page.navigate", {"url": "https://example.com"})
client.finalize_tabs([])
client.close()
```

## Go SDK Pattern

```go
package main

import (
	"fmt"
	"log"
	"time"

	obu "github.com/Kiana-ZY/open-browser-use-windows/packages/open-browser-use-go"
)

func main() {
	browser, err := obu.ConnectActive(obu.Options{
		SessionID: "obu-issue-scan-20260510",
		Timeout:   20 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()
	defer browser.Client.FinalizeTabs(nil)

	if _, err := browser.Client.NameSession("Issue scan - OBU"); err != nil {
		log.Fatal(err)
	}
	tab, err := browser.NewTab()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tab.Goto("https://example.com", obu.GotoOptions{
		WaitUntil: obu.LoadStateDOMContentLoaded,
		Timeout:   15 * time.Second,
	}); err != nil {
		log.Fatal(err)
	}
	title, err := tab.Title()
	if err != nil {
		log.Fatal(err)
	}
	info, err := tab.PageInfo(obu.TextOptions{MaxChars: 2000})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(title, info.URL)
}
```

Use the low-level client when you need raw JSON-RPC/CDP calls:

```go
client := obu.NewClient(obu.Options{
	SocketPath: "127.0.0.1:19832",
	SessionID:  "obu-docs-scan-20260510",
})
defer client.Close()

tab, err := client.CreateTab()
if err != nil {
	log.Fatal(err)
}
tabID := int(tab.(map[string]any)["id"].(float64))
if _, err := client.ExecuteCDP(tabID, "Page.navigate", obu.Params{"url": "https://example.com"}); err != nil {
	log.Fatal(err)
}
_, _ = client.FinalizeTabs(nil)
```

## Core Methods

Common Browser Use JSON-RPC methods:

- `ping`
- `getInfo`
- `createTab`
- `getTabs`
- `getUserTabs`
- `getUserHistory`
- `claimUserTab`
- `finalizeTabs`
- `nameSession`
- `attach`
- `detach`
- `executeCdp`
- `moveMouse`
- `waitForFileChooser`
- `setFileChooserFiles`
- `waitForDownload`
- `downloadPath`
- `readClipboardText`
- `writeClipboardText`
- `readClipboard`
- `writeClipboard`
- `turnEnded`

High-level SDK tab helpers mirror the CLI/MCP object protocol:

- JavaScript: `tab.pageInfo`, `tab.text`, `tab.snapshot`, `tab.screenshot`, `tab.click`, `tab.fill`
- Python: `tab.page_info`, `tab.text_result`, `tab.snapshot`, `tab.screenshot_result`, `tab.click`, `tab.fill`
- Go: `tab.PageInfo`, `tab.Text`, `tab.Snapshot`, `tab.Screenshot`, `tab.Click`, `tab.Fill`

The older convenience helpers such as `domSnapshot` / `dom_snapshot` and
locator inner-text remain available for string-first workflows.

CLI unrestricted call:

```sh
open-browser-use call --session-id "$OBU_SESSION_ID" --method getInfo --params '{}'
open-browser-use call --session-id "$OBU_SESSION_ID" --method executeCdp --params '{"target":{"tabId":123},"method":"Runtime.evaluate","commandParams":{"expression":"document.title"}}'
```

CLI action plan:

```sh
export OBU_SESSION_ID="obu-docs-scan-$(date +%Y%m%d%H%M%S)"
open-browser-use run --session-id "$OBU_SESSION_ID" -c '
name-session "Docs scan - OBU"
open-tab https://docs.browser-use.com
wait-load domcontentloaded
page-info
finalize-tabs []
'
```

The action plan format is intentionally small: one action per line, comments
with `#`, shell-like quotes, shared session/turn, and a default tab set by
`open-tab` or `claim-tab`. Supported actions include `ping`, `info`, `tabs`,
`user-tabs`, `history`, `name-session`, `open-tab`, `claim-tab`, `navigate`,
`wait-load`, `page-info`, `text`, `snapshot`, `screenshot`, `click`, `fill`,
`cdp`, `move-mouse`, `wait-file-chooser`, `set-file-chooser-files`,
`finalize-tabs`, `turn-ended`, and `call`.

Add `--trace-log <path>` to write a JSONL audit trail for action plans. Each
entry includes `timestamp`, `sessionId`, `turnId`, `line`, `action`, `risk`,
`tabId`, `durationMs`, `ok`, and `error` when a step fails.

## MCP Server

Use the stdio MCP server when the surrounding runtime supports local MCP tools:

```toml
[mcp_servers.open_browser_use]
command = "obu"
args = ["mcp", "--session-id", "obu-<task-or-conversation-id>"]
```

`obu mcp` speaks newline-delimited JSON-RPC on stdin/stdout. It handles
`initialize`, `ping`, `tools/list`, and `tools/call`, and exposes tools that
mirror the CLI action surface:

- `ping`, `info`, `tabs`, `user_tabs`, `history`
- `open_tab`, `claim_tab`, `navigate`, `wait_load`, `page_info`
- `text`, `snapshot`, `screenshot`, `click`, `fill`
- `cdp`, `move_mouse`, `wait_file_chooser`, `set_file_chooser_files`
- `name_session`, `finalize_tabs`, `turn_ended`, `call`, `run_action_plan`

Pass `--socket` or `--socket-dir` in the MCP `args` only when the runtime needs
an explicit Open Browser Use relay. Otherwise the server uses the same relay
discovery as the CLI. Pass a fresh `--session-id` for each agent task or
conversation.

Pass `--trace-log <path>` in the MCP `args` to trace direct MCP tool calls with
the same JSONL format as action plans. The `run_action_plan` MCP tool writes
per-action trace rows through the runner and does not add a duplicate wrapper
row.

For the common agent-facing paths, prefer the same stable object shapes from
both CLI `--json` and MCP `structuredContent`:

- `ping` -> `{ "status": "pong" }`
- `tabs` / `user_tabs` / `history` -> `{ "items": [...] }`
- `open_tab` -> `{ "tab": ..., "navigate": ...? }`
- `claim_tab` -> `{ "tab": ... }`
- `navigate` -> `{ "navigate": ... }`
- `page_info` -> `{ "title": ..., "url": ..., "readyState": ..., "text": ... }`
- `text` -> `{ "text": ... }`
- `snapshot` -> `{ "items": [...] }`
- `screenshot` -> `{ "path": ..., "bytes": ..., "format": "png", "clip": ...? }`
- `click` / `fill` -> `{ "ok": true, "action": ..., "ref": ... }`
- `wait_load` -> `{ "readyState": ... }`
- `name_session` / `finalize_tabs` / `turn_ended` -> `{ "ok": true }`

This keeps agent integrations from branching on transport-specific wrappers.

## Observability And Risk Labels

The CLI and MCP server add `request_id` to browser JSON-RPC params when callers
do not provide one. Action plans and traced MCP tools also label actions:

- `read`: `ping`, `info`, `tabs`, `user-tabs`, `history`, `wait-load`, `page-info`, `text`, `snapshot`, `screenshot`
- `navigation`: `open-tab`, `claim-tab`, `navigate`
- `interaction`: `click`, `fill`, `move-mouse`
- `file-system`: `set-file-chooser-files`
- `session`: `name-session`, `turn-ended`, `finalize-tabs`
- `unrestricted`: `cdp`, `call`

OBU records these labels for auditing and debugging. Higher-level runtimes are
still responsible for confirmation policy, especially before uploads, downloads,
clipboard access, form submission, purchases, deletion, external message
sending, or unrestricted CDP/RPC calls with side effects.

SDK request escape hatch:

```ts
await browser.client.request("executeCdp", {
  target: { tabId: 123 },
  method: "Runtime.evaluate",
  commandParams: { expression: "document.title" },
});
```

```py
browser.client.request("executeCdp", {
    "target": {"tabId": 123},
    "method": "Runtime.evaluate",
    "commandParams": {"expression": "document.title"},
})
```

```go
_, err := browser.Client.Request("executeCdp", obu.Params{
	"target":        obu.Params{"tabId": 123},
	"method":        "Runtime.evaluate",
	"commandParams": obu.Params{"expression": "document.title"},
})
```

## User Tab Claiming

1. List open user tabs with `open-browser-use user-tabs --session-id "$OBU_SESSION_ID"` or SDK `getUserTabs`.
2. Select the tab from returned data using visible evidence: title, URL, recency, and group.
3. Claim it with `open-browser-use claim-tab --session-id "$OBU_SESSION_ID" --tab-id <id>` or SDK `claimUserTab` / `claim_user_tab` / `ClaimUserTab`.
4. Use the returned controllable tab for later commands.

Never invent or reuse stale tab ids.

## Tab Cleanup

Before ending browser work, finalize exactly once for the active session:

```sh
open-browser-use finalize-tabs --session-id "$OBU_SESSION_ID" --keep '[]'
```

Omit tabs by default. Keep a tab only when the user needs that live page after
the turn. Use `status: "deliverable"` for a user-facing output or requested
open page. Use `status: "handoff"` only when the task is still in progress and
the user or a later turn should continue from the current task group, such as a
page waiting for login, approval, payment, CAPTCHA, or other user input.

Treat finalization as the last Open Browser Use browser action of the turn. If
more browser work is needed, do it before finalizing, then finalize once with
the final tab disposition.

## File Chooser Pattern

1. Start waiting with `wait-file-chooser --tab-id <id>` or SDK `waitForFileChooser` / `wait_for_file_chooser` / `WaitForFileChooser`.
2. Trigger the file picker in the page, usually through a click driven by CDP or a higher-level automation layer.
3. Set absolute file paths:

```sh
open-browser-use set-file-chooser-files --file-chooser-id <id> --file /absolute/path/file.txt
```

Use repeated `--file` values or comma-separated paths for multiple files.
