# Open Browser Use Python SDK

Python client for controlling a real Chrome profile through Open Browser Use.
The package distribution is `open-browser-use-sdk`; the import module remains
`open_browser_use`.

## Installation

```sh
pip install open-browser-use-sdk
```

The SDK expects the `open-browser-use` CLI and Chrome extension to already be
installed and connected:

```sh
open-browser-use ping
open-browser-use info
```

## Usage

```py
import json
from pathlib import Path

from open_browser_use import connect_open_browser_use

registry = json.loads(Path("/tmp/open-browser-use/active.json").read_text())

browser = connect_open_browser_use(
    socket_path=registry["socketPath"],
    session_id="python-sdk-example",
)

try:
    browser.client.name_session("Python SDK example - OBU")
    tab = browser.new_tab()
    tab.goto("https://example.com", wait_until="domcontentloaded")
    print(tab.title())

    info = tab.page_info(max_chars=2000)
    text = tab.text_result(selector="main", max_chars=2000)
    snapshot = tab.snapshot(limit=50)
    screenshot = tab.screenshot_result(selector="main")

    tab.click("@1")
    tab.fill("@2", "hello")
finally:
    browser.client.finalize_tabs([])
    browser.close()
```

Use `OpenBrowserUseClient` directly when you need raw Browser Use JSON-RPC or
CDP methods.

The structured helpers mirror CLI `--json` and MCP `structuredContent`:
`page_info`, `text_result`, `snapshot`, `screenshot_result`, `click`, and
`fill`. The legacy convenience `tab.text()` still returns a string.
