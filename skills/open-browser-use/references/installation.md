# Open Browser Use Installation

Read this reference when the user asks to install, verify, repair, or explain Open Browser Use setup.

## Components

- Browser extension (MV3): the browser-side controller. Works with Chrome and Edge. Installing or enabling it may require the user to approve browser prompts.
- Native host and CLI: the local `open-browser-use` binary, also exposed as `obu` when installed from supported packages.
- SDKs: JavaScript, Python, and Go clients that connect to the active native host relay.

## Install The CLI

Use one of the supported package routes:

```sh
npm install -g open-browser-use
```

Homebrew publishing is repository-specific. If a tap is configured for this
fork, install from that tap; otherwise use npm, a GitHub Release binary, or a
local build.

Verify:

```sh
open-browser-use version
obu version
```

If the short alias is unavailable, use `open-browser-use`.
Running `open-browser-use` with no subcommand prints the CLI version, browser
extension detection status, extension version when available, and the next setup
or upgrade command.

## Set Up The Browser

After installing the CLI, register the native messaging host and install the extension:

```sh
# For Chrome (default)
open-browser-use setup

# For Edge
open-browser-use setup --browser edge
```

The browser may ask the user to confirm or enable the Open Browser Use extension. Do not bypass this user step.

While the Chrome Web Store item is unavailable or pending review, use the release ZIP path:

```sh
# For Chrome
open-browser-use setup beta

# For Edge
open-browser-use setup beta --browser edge
```

This downloads the latest keyed `open-browser-use-chrome-extension-*.zip` from GitHub Releases, registers the native host for that stable extension id, opens `chrome://extensions/` or `edge://extensions/`, and reveals the ZIP in Finder or the system file manager. Ask the user to enable Developer mode and drag that ZIP into the browser extensions page.

Repair only the native host manifest:

```sh
open-browser-use install-manifest --browser edge  # or omit for Chrome
```

Print the manifest without installing:

```sh
open-browser-use manifest
```

## Platform Notes

- On Windows, pass `--browser edge` to configure Microsoft Edge instead of Chrome.
- macOS and Windows can require the user to approve or enable the extension after the browser sees it.
- Linux external extension registration can require elevated permissions depending on installation paths.
- Native messaging host name is `com.ifuryst.open_browser_use.extension`; this
  compatibility identifier is kept stable even though the project name is Open
  Browser Use for Windows.
- On Unix-like systems, the default socket is under `/tmp/open-browser-use/`. On Windows, the relay uses TCP on `127.0.0.1:19832`.

## Verification

Run:

```sh
open-browser-use ping --session-id "$OBU_SESSION_ID"
open-browser-use info --session-id "$OBU_SESSION_ID"
open-browser-use user-tabs --session-id "$OBU_SESSION_ID"
```

For one-off installation checks, a temporary session id is enough. Agent browser
tasks should still create and reuse a task-unique session id before opening or
claiming tabs.

If `ping` cannot communicate with the browser, ask the user whether the browser (Chrome or Edge) is installed and running, whether the extension is enabled, and whether they approved any browser prompt. Then use [troubleshooting.md](troubleshooting.md).
