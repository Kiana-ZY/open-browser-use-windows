## [2026-05-11 15:26] | Task: Browser use SDK parity

### Execution Context

- Agent ID: `Codex`
- Base Model: `GPT-5`
- Runtime: `Codex Desktop / PowerShell / Windows`

### User Query

> 继续推进 Open Browser Use，让它成为 Codex app browser use 的平替甚至更优方案。

### Changes Overview

Scope: JavaScript SDK, Python SDK, Go SDK, SDK docs, execution plan.

Key Actions:

- Added structured tab helpers across SDKs for page info, bounded text,
  interactive snapshots, screenshots, click, and fill.
- Kept existing string-first helpers such as `domSnapshot` / `dom_snapshot`
  and locator inner text compatible.
- Added JS, Python, and Go fake-socket tests for the new structured helper
  shapes and CDP call flow.
- Made Python SDK tests Windows-compatible by allowing `host:port` socket paths
  and using TCP fake sockets in tests.
- Documented the SDK parity surface in package READMEs and the agent protocol
  reference.

### Design Intent (Why)

M6 should make CLI, MCP, and SDK integrations speak the same agent-facing object
contract without making the SDKs large browser engines. Thin helpers keep the
escape hatch of raw CDP/RPC while letting common agent workflows avoid repeated
DOM expression glue.

### Files Modified

- `packages/open-browser-use-js/src/index.ts`
- `packages/open-browser-use-js/test/frame.test.mjs`
- `packages/open-browser-use-js/README.md`
- `packages/open-browser-use-python/open_browser_use/client.py`
- `packages/open-browser-use-python/test_client.py`
- `packages/open-browser-use-python/README.md`
- `packages/open-browser-use-go/browser.go`
- `packages/open-browser-use-go/client_test.go`
- `packages/open-browser-use-go/README.md`
- `skills/open-browser-use/references/sdk-and-protocol.md`
- `docs/exec-plans/active/2026-05-11-browser-use-parity-plus.md`

### Validation

- `go test ./...`
- `D:\work\miniconda\python.exe -m py_compile open_browser_use\client.py test_client.py`
- `D:\work\miniconda\python.exe -m unittest test_client`
- `pnpm --dir packages\browser-client-rewrite test`

Note: `pnpm --dir packages\open-browser-use-js test` still requires installing
workspace Node dependencies because local `node_modules` is absent.
