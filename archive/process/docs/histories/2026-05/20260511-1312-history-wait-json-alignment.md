## [2026-05-11 13:12] | Task: align history and wait json shapes

### 🤖 Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### 📥 User Query

> 继续

### 🛠 Changes Overview

**Scope:** `cmd/open-browser-use`, `packages/open-browser-use-cli`, `skills/open-browser-use`, `docs/histories`

**Key Actions:**

- **[CLI]**: Normalized `history --json` to return `{ "items": [...] }` and `wait --json` to return `{ "readyState": ... }`.
- **[Test]**: Added command-level tests that lock both result shapes for machine-readable use.
- **[Docs]**: Extended the README and repo-local skill so the common object-shaped `--json` contract now covers page summary, raw text, snapshots, history, and wait state reads.

### 🧠 Design Intent (Why)

After aligning `page-info`, `text`, and `snapshot`, the remaining high-frequency
read commands still exposed mixed shapes. Bringing `history` and `wait` into
the same object-oriented pattern reduces agent-side branching further and makes
the CLI feel more like one coherent protocol.

### 📁 Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `docs/histories/2026-05/20260511-1312-history-wait-json-alignment.md`
