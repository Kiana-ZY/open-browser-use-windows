## [2026-05-11 12:55] | Task: add OBU vs Codex browser comparison

### 🤖 Execution Context

- **Agent ID**: `Codex`
- **Base Model**: `GPT-5`
- **Runtime**: `Codex desktop`

### 📥 User Query

> 使用我的 open browser use 来感受一下，速度 tokens消耗 以及和codex app的 browser use比有什么优劣，还有的这个项目与skill还能有什么改进？先来对比文档

### 🛠 Changes Overview

**Scope:** `docs/wiki/browser-client`, `docs/histories`

**Key Actions:**

- **[Comparison Doc]**: Added a reusable comparison note between Open Browser Use and Codex Browser Use, covering speed, token-cost shape, real-browser vs in-app-browser tradeoffs, interaction model differences, and product-surface gaps.
- **[Evidence Split]**: Marked which conclusions came from direct local OBU smoke checks and which remained architectural inference because the Codex in-app browser route did not finish a same-session benchmark.
- **[Recommendations]**: Captured concrete repository follow-ups, including CLI/MCP/skill vocabulary unification, bounded extraction flags, richer interaction fallback, and explicit routing guidance.

### 🧠 Design Intent (Why)

The user asked for a practical comparison rather than a code-only change. The
most useful repo artifact is a durable note that future agents and maintainers
can read when deciding when OBU should be preferred over Codex Browser Use and
which product gaps matter most. Writing it into the repository also keeps the
comparison from being trapped in transient chat context.

### 📁 Files Modified

- `docs/wiki/browser-client/notes/open-browser-use-vs-codex-browser-use.md`
- `docs/histories/2026-05/20260511-1255-obu-vs-codex-browser-use-comparison.md`
