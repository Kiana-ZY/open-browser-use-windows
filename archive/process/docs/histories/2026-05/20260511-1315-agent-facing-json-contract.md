## [2026-05-11 13:15] | Task: stabilize agent-facing json contract

### 🤖 Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### 📥 User Query

> 可以，然后补齐这个

### 🛠 Changes Overview

**Scope:** `cmd/open-browser-use`, `packages/open-browser-use-cli`, `skills/open-browser-use`, `docs/histories`

**Key Actions:**

- **[CLI]**: Added a normalization layer so common direct CLI commands return stable object-shaped `--json` results instead of leaking raw JSON-RPC result diversity.
- **[Behavior]**: Standardized action-command JSON output for `ping`, `claim-tab`, `navigate`, `name-session`, `finalize-tabs`, and `turn-ended`.
- **[Test]**: Added command-level tests that lock the normalized object shapes for representative action commands.
- **[Docs]**: Extended the CLI README and repo-local OBU skill so the stable agent-facing JSON contract is visible in one place.

### 🧠 Design Intent (Why)

The repository had already improved high-frequency read commands, but the rest
of the CLI surface still exposed mixed raw result shapes. That forces agent
wrappers to branch on command-specific transport details. Normalizing the
high-frequency direct CLI commands into a compact object protocol makes the CLI
feel more like a coherent agent contract rather than a thin RPC passthrough.

### 📁 Files Modified

- `cmd/open-browser-use/main.go`
- `cmd/open-browser-use/main_test.go`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/SKILL.md`
- `docs/histories/2026-05/20260511-1315-agent-facing-json-contract.md`
