# Open Browser Use vs Codex Browser Use

This note compares two browser-control surfaces that matter in this repository:

- Open Browser Use (OBU): this project's real-browser Chrome/Edge route
- Codex Browser Use: the bundled Codex browser runtime, especially its in-app
  browser (IAB) route

The goal is not to declare a universal winner. The useful question is which
surface is better for a given agent task, and what that implies for the OBU
product surface and skill guidance.

## Scope And Evidence

This note mixes two evidence levels:

- Direct local OBU smoke checks performed in this repository environment on
  2026-05-11
- Repository and reverse-engineering evidence for Codex Browser Use capability
  shape, backend routing, and safety model

The Codex in-app browser route was not fully benchmarked end-to-end in the same
 session because runtime initialization timed out twice during this task. Claims
about Codex Browser Use speed below should therefore be read as architectural
inference unless marked as OBU smoke data.

## Short Take

Use Open Browser Use when you need the user's real browser state, want a small
shell/MCP-friendly control surface, and care about keeping browser-action token
cost low by default.

Use Codex Browser Use when the task is interaction-heavy, UI-sensitive, or
needs richer runtime semantics such as structured tab objects, locator APIs,
screenshots, guarded permissions, and stronger session-aware policy.

## What Was Measured For OBU

The following OBU CLI actions were exercised successfully:

- `obu ping`
- `obu info`
- `obu open-tab`
- `obu wait`
- `obu text`
- `obu snapshot`
- `obu finalize-tabs`

Observed timings from the local environment:

| Target | open-tab | wait | text | snapshot | finalize-tabs |
| --- | ---: | ---: | ---: | ---: | ---: |
| `https://example.com` | ~699 ms | ~916 ms | ~80 ms | ~62 ms | ~79 ms |
| `https://news.ycombinator.com` | ~2299 ms | ~281 ms | ~279 ms | not timed in summary run | ~67 ms |

Observed response shape:

- `text` returned a narrow JSON object with one large string payload
- `snapshot` returned a compact array of interactive elements with index, tag,
  text, href, and selector hints

That confirms the practical strength of OBU's current CLI surface: one command,
one narrow result, minimal wrapper overhead.

## Speed And Token Cost

### OBU Strengths

OBU feels fast because its default agent-facing actions are narrow:

- `text` returns page text directly
- `snapshot` returns a flat list of interactable items
- `click` and `fill` are lightweight follow-up actions

For shell-first or MCP-first agents, that keeps both transport overhead and
model context overhead low. The agent does not need to carry a large Playwright
session object or repeated browser-runtime scaffolding in every step.

### OBU Limits

OBU is token-efficient by default, not absolutely.

- `text` can become expensive on long article pages or large list pages because
  it returns broad body text
- `snapshot` can become expensive on link-dense pages because it emits many
  candidate elements

In other words, OBU saves tokens mostly by choosing a small result schema, not
because it has magical compression.

### Codex Browser Use Tradeoff

Codex Browser Use usually carries more runtime structure:

- browser object
- tab object
- richer Playwright-style methods
- screenshots and DOM snapshots as first-class tools
- policy and permission handling in the runtime

That generally raises control-surface complexity and sometimes cold-start cost,
but pays back on harder tasks because the model spends fewer turns stitching raw
browser primitives together.

## Real Browser vs In-App Browser

This is the biggest architectural difference.

### Open Browser Use

OBU is built around the user's real Chrome or Edge profile:

- existing tabs can be claimed
- history can be queried
- the user's login state and cookies are naturally reused
- tabs live in the user's normal browser workspace

That is ideal for:

- continuing work from an already-open page
- authenticated SaaS workflows
- browser history lookup
- tasks where the user expects to inspect or keep the resulting tab

### Codex Browser Use

Codex Browser Use supports multiple backend types. In this repository's current
knowledge model, the in-app browser route is a separate runtime surface with its
own backend selection and session ownership rules.

That is ideal for:

- isolated local-app testing
- controlled verification of UI state
- interaction sequences that benefit from a richer runtime abstraction
- tasks where stronger runtime mediation is more valuable than real-profile
  continuity

## Interaction Model

### OBU Today

OBU's CLI interaction layer is intentionally thin.

The current `snapshot`, `click`, and `fill` commands are implemented through
CDP `Runtime.evaluate` with a ref-based DOM convention:

- `snapshot` evaluates page JavaScript and stores a compact element list
- `click` resolves `data-obu-ref` and calls `.click()`
- `fill` resolves `data-obu-ref`, assigns `value`, and dispatches an input event

See:

- `cmd/open-browser-use/main.go`

This is great for speed and simplicity, but it is less robust than a richer
interaction stack on:

- complex controlled inputs
- shadow DOM
- iframes
- custom widgets with nontrivial event sequencing
- pages where "DOM click" differs from a realistic user gesture

### Codex Browser Use

Codex Browser Use exposes a broader interaction stack:

- tab lifecycle
- Playwright-like locators
- CDP
- CUA and DOM-CUA paths
- screenshots
- tab-scoped clipboard and dev logs

That makes it heavier, but also more capable for interaction-heavy work and UI
verification.

## Safety And Policy

OBU is intentionally open-ended. The repository architecture states that the SDK
does not embed Codex-style site restrictions, turn policy, or confirmation
logic, and recommends implementing such policy in the upper-layer runtime when
needed.

Codex Browser Use is stronger on this axis. Its runtime model includes:

- backend-aware capability routing
- site policy checks
- confirmation handling for higher-risk browser actions
- session ownership checks for the in-app browser route

For production agent use, this is one of the main reasons Codex Browser Use can
feel "heavier but safer."

## Decision Guide

Choose OBU first when:

- you need the user's real browser profile
- the user already has the right tab open
- login continuity matters
- the workflow is mostly open, read, claim, navigate, click a few things, and
  hand back a tab
- shell CLI or MCP is the integration surface
- minimizing token cost per browser step matters

Choose Codex Browser Use first when:

- the task is mostly within localhost or an isolated verification surface
- you need locator-rich interaction and repeated UI assertions
- screenshots are part of the normal control loop
- runtime policy and confirmation behavior matter
- the interaction may involve many state transitions where a richer browser
  abstraction reduces agent fragility

## Current Product Gaps In OBU

The comparison surfaced several practical gaps worth treating as product work,
not just documentation cleanup.

### 1. Product Surface Drift

The repo-local OBU skill emphasizes:

- `snapshot`
- `text`
- `click`
- `fill`

But the MCP and action-runner surface emphasizes:

- `open_tab`
- `navigate`
- `wait_load`
- `page_info`
- `cdp`
- `run_action_plan`

Those are both valid, but they feel like two different products. Agent guidance
is currently split between a lightweight CLI-first mental model and a managed
tab/MCP mental model.

### 2. Naming And JSON Shape Inconsistency

The direct CLI and MCP surfaces do not expose the same nouns consistently.

Example:

- the action runner and MCP expose `page_info`
- the direct CLI primarily exposes `text`

The underlying capabilities overlap, but the naming difference creates
avoidable friction for agents and wrappers.

### 3. Token-Aware Extraction Controls

The current OBU strength is narrow output, but there is little built-in control
for bounded extraction. Useful additions would include:

- `text --max-chars`
- `text --selector`
- `snapshot --limit`
- `snapshot --interactive-only`
- structured page summary modes that return title, URL, and a bounded excerpt

### 4. Richer Interaction Fallbacks

The thin `click` and `fill` implementation is fast, but OBU would benefit from
an explicit "lightweight first, richer fallback second" interaction ladder.

### 5. Explicit Routing Guidance

The repository would benefit from a short guide that tells agent authors when to
pick OBU over Codex Browser Use and when not to.

## Recommendations

### Near-Term

1. Unify agent-facing docs so CLI, MCP, and skill vocabulary tell one story.
2. Add a direct CLI `page-info` command, or else rewrite docs to consistently
   prefer `text`.
3. Add bounded extraction flags for `text` and `snapshot`.
4. Document a recommended routing rule: "real user browser -> OBU; isolated app
   verification -> Codex Browser Use."

### Medium-Term

1. Add a benchmark script that records latency and payload size for common
   browser tasks.
2. Add a more robust interaction tier for complex inputs and clicks.
3. Expose a normalized cross-surface JSON schema for page summary and element
   snapshots.

## Bottom Line

Open Browser Use is better when the job is "use the user's actual browser
cheaply and directly."

Codex Browser Use is better when the job is "drive a browser runtime with more
guardrails, richer interaction semantics, and better support for complex UI
work."

The most important next step for this repository is not adding yet another
browser primitive. It is tightening the agent-facing product surface so CLI,
MCP, skill, and documentation describe one coherent operating model.
