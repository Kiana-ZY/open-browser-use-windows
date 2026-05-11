## [2026-05-11 17:58] | Task: project rename, archive cleanup, and local build

### Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop on Windows / PowerShell + WSL bash`

### User Query

> Rename the project to Open Browser Use for Windows, keep the short alias `obu`,
> update the version, organize non-core local project files into categorized
> folders, and upload the result to GitHub. Use the local project build, not an
> online package version.

### Changes Overview

**Scope:** repository metadata, extension, CLI, SDKs, docs, archive layout, CI,
release packaging, and agent skill docs.

**Key Actions:**

- **Project identity**: renamed user-facing project docs and package metadata to
  Open Browser Use for Windows while preserving the `obu` shortcut and the
  compatibility native messaging host id.
- **Version bump**: updated active packages, extension manifest, CLI, SDKs, and
  release notes to `0.1.29`.
- **Archive cleanup**: moved old process docs, histories, doc site, research
  references, prototype packages, and local agent snapshots under categorized
  `archive/` paths.
- **Local build hardening**: added shared runtime tool discovery and a Node-only
  ZIP helper so Windows/WSL local packaging does not depend on online packages
  or unavailable Unix `zip` / `unzip` commands.
- **Verification**: ran local CI, release packaging, Go tests, JS tests/build,
  Python SDK tests, docs/hygiene checks, action pinning checks, script syntax
  checks, version checks, and stale-name scans.

### Design Intent (Why)

The repository should now read as a focused Windows-first browser automation
project instead of a template or research workspace. Active code paths stay
small and buildable, while historical and research material remains available
for traceability without being runtime dependencies.

### Files Modified

- `README.md`
- `README.zh-CN.md`
- `AGENTS.md`
- `docs/ARCHITECTURE.md`
- `docs/CODEX_AND_CLAUDE_USAGE.md`
- `docs/CHROME_WEB_STORE_RELEASE.md`
- `docs/releases/feature-release-notes.md`
- `apps/chrome-extension/manifest.json`
- `cmd/open-browser-use/main.go`
- `packages/open-browser-use-js/src/index.ts`
- `packages/open-browser-use-python/open_browser_use/client.py`
- `packages/open-browser-use-go/client.go`
- `scripts/runtime-tools.sh`
- `scripts/zip-tool.mjs`
- `scripts/ci.sh`
- `scripts/release-package.sh`
- `scripts/package-chrome-extension.sh`
- `scripts/package-skill.sh`
- `archive/README.md`
