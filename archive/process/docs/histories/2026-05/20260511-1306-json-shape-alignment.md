## [2026-05-11 13:06] | Task: align CLI json result shapes

### 🤖 Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### 📥 User Query

> 继续把

### 🛠 Changes Overview

**Scope:** `cmd/open-browser-use`, `packages/open-browser-use-cli`, `skills/open-browser-use`, `docs/histories`

**Key Actions:**

- **[CLI]**: Normalized the `--json` shapes of the high-frequency read commands so direct agent consumption is more consistent.
- **[Behavior]**: `text --json` now returns `{ "text": ... }` and `snapshot --json` now returns `{ "items": [...] }`, matching the object-oriented style already used by `open-tab --json` and `page-info --json`.
- **[Test]**: Added command-level tests that lock the new `text` and `snapshot` JSON result shapes.
- **[Docs]**: Updated the CLI README and repo-local OBU skill to document the stable object shapes for common machine-readable commands.

### 🧠 Design Intent (Why)

Before this change, common OBU read commands exposed mixed JSON shapes: some
returned raw arrays, some returned nested CDP values, and some returned
top-level objects. That inconsistency adds unnecessary adapter code for agents.
Normalizing the top commands to object-shaped results reduces branching in agent
wrappers while keeping the payloads small.

### 📁 Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `docs/histories/2026-05/20260511-1306-json-shape-alignment.md`
