# Security Policy

Open Browser Use for Windows connects automation clients to the user's real
Chrome or Microsoft Edge profile through a Chromium extension, Native
Messaging, and a local native host. Treat it as local browser control
infrastructure, not as a sandbox boundary.

## Supported Scope

Security fixes are expected to cover the current Chromium route:

- MV3 Chrome/Edge extension under `apps/chrome-extension/`.
- Go native host and CLI under `cmd/open-browser-use/` and `internal/`.
- JavaScript, Python, and Go SDK packages under `packages/`.
- Installer, manifest, release, and skill documentation that affects how users enable browser control.

Experimental references, reverse-engineering notes, and archived planning material are useful context, but they are not independently supported interfaces unless they are wired into the current route.

## Reporting Vulnerabilities

If you find a security issue, do not open a public issue with exploit details. Report it privately to the maintainers first.

Please include:

- A short description of the impact.
- Affected component and version, commit, or release.
- Reproduction steps or proof of concept, with secrets and personal browsing data removed.
- Whether the issue requires local access, extension installation, a malicious web page, or a malicious automation client.

Do not include real credentials, private cookies, personal browsing history, or unredacted local paths in reports. Maintainers should coordinate disclosure timing before publishing technical details.

## Security Boundaries

The Chromium route intentionally exposes powerful browser automation primitives
to local callers that can reach the Open Browser Use relay.

- The SDKs and CLI do not implement a Codex-style site policy, command allowlist, or user approval workflow.
- Upper-layer runtimes are responsible for deciding which sites, methods, downloads, clipboard operations, and destructive actions are allowed.
- The native host and extension are not a general-purpose sandbox for untrusted code.
- Any integration that accepts remote instructions must add its own authentication, authorization, audit logging, and confirmation policy before invoking Open Browser Use for Windows.

## Local Transport

The Go native host bridges Chromium Native Messaging stdio to a local client
relay.

- On Windows, the relay listens on `127.0.0.1:19832`.
- On Unix-like systems, the default socket registry is under
  `/tmp/open-browser-use/`, `active.json` is written with `0600` permissions,
  and socket files are created with the current user's local permissions.
- The CLI may scan the socket directory on Unix-like systems when the active
  registry is missing or stale.
- Callers should treat access to the user's local account as access to the
  browser automation channel.

Planned hardening includes client tokens, peer validation, safer stale-socket handling, and security-focused failure-path tests.

## Extension Permissions

The MV3 extension uses high-privilege Chromium APIs because browser control
requires them:

- `chrome.debugger` for CDP access.
- `tabs` and `tabGroups` for tab and session management.
- `history` for history lookup.
- `downloads` for download observation.
- Native Messaging to communicate with the local host.

Users must understand that installing and enabling the extension allows Open
Browser Use for Windows to inspect and operate real Chrome or Edge tabs in
their active profile. The native messaging host manifest must keep
`allowed_origins` restricted to the expected Open Browser Use extension ID.

## Clipboard, Downloads, And Files

Clipboard, download, and file chooser features are sensitive.

- Clipboard helpers run through the controlled page context and should only be called for explicit user workflows.
- Download paths and file chooser paths can reveal private filesystem structure; do not log them unless needed, and redact them from shared reports.
- Integrations should confirm uploads, downloads, file selection, form submission, purchases, deletion, and external message sending before execution.

## Action Auditing

Open Browser Use for Windows adds `request_id` to browser JSON-RPC params when
callers do not provide one. CLI action plans and MCP tool calls can also write
JSONL audit rows with `--trace-log <path>`.

Trace rows include session id, turn id, action, risk class, tab id, duration,
success status, and error text. Risk classes are intentionally coarse:

- `read` for diagnostics, page state, text, snapshots, screenshots, tabs, history, and ping/info.
- `navigation` for opening, claiming, and navigating tabs.
- `interaction` for click, fill, and cursor movement.
- `file-system` for setting file chooser paths.
- `session` for session naming, turn end, and tab finalization.
- `unrestricted` for raw CDP and generic RPC calls.

These labels are an observability surface, not an authorization boundary.
Upper-layer runtimes must still enforce their own confirmation policy before
externally visible or destructive actions.

## Secrets And Data Handling

- Never commit credentials, cookies, API keys, browser profile data, native host manifests with private machine-specific paths, or captured user data.
- Keep examples synthetic and minimal.
- Redact local usernames, home directories, tokens, session IDs, and private browsing history from docs, history entries, release notes, and issue reports.
- Prefer environment variables or local config files ignored by Git for development secrets.

## Supply Chain

Repository-level dependency, SBOM, provenance, and release artifact expectations live in `docs/SUPPLY_CHAIN_SECURITY.md`. Security-sensitive dependency or release changes should update that document in the same change when expectations move.

## Maintainer Checklist

For security-affecting changes, verify:

- Manifest origins remain constrained to the intended extension.
- Native host install paths and generated manifests do not leak private local state.
- Relay ports, socket files, registries, temp files, and release artifacts use the intended permissions.
- Tests cover authorization, stale transport state, and failure paths where practical.
- User-facing setup docs describe the real permissions and risks.
