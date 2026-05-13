# CloakBrowser Comparison

This note captures what Open Browser Use for Windows should and should not
borrow from CloakBrowser after reviewing `CloakHQ/CloakBrowser` at commit
`ad4d946ca6c5a1490223818e21e8241910deed8b` on 2026-05-13.

## Summary

CloakBrowser is a patched Chromium distribution with Python and JavaScript
wrappers that position themselves as drop-in Playwright/Puppeteer replacements.
Its product strength is not only the browser binary. It has a very clear
developer funnel: install, first-run binary management, visible diagnostics,
integration examples, Docker entrypoints, a changelog, platform matrix, and
specific troubleshooting paths.

Open Browser Use for Windows has a different security boundary. It controls the
user's real Chrome or Edge profile through a local extension and native host.
It should not borrow anti-detection, CAPTCHA-bypass, fingerprint spoofing, proxy
rotation, or stealth claims. Those would push the project away from its stated
local browser-control infrastructure boundary.

## Useful Patterns To Borrow

| CloakBrowser pattern | Why it matters | OBU action |
| --- | --- | --- |
| One-command install and visible first-run status | Users know whether the runtime is installed before writing code. | Keep `obu doctor` prominent and make `doctor --browser all` the broad preflight. |
| Stable wrapper APIs across Python and JavaScript | Framework users can switch with little glue code. | Keep CLI `--json`, MCP `structuredContent`, and SDK helpers aligned. |
| Platform matrix in user-facing docs | Users can quickly see which OS/browser path is expected to work. | Maintain Chrome/Edge plus Windows TCP and Unix socket compatibility notes. |
| Integration examples for agent frameworks | Adoption is easier when runtime-specific examples are concrete. | Keep Codex/Claude skill, MCP config, and SDK examples in the same contract. |
| Clear changelog and release notes | Users can decide whether to upgrade and how to debug regressions. | Add feature-release-note entries for diagnostic and agent workflow changes. |
| Troubleshooting by symptom | Users recover faster from setup and runtime failures. | Route ping/setup failures through `doctor --browser all --json` and documented next steps. |
| Local server security warning | Browser-control endpoints are powerful. | Keep localhost-only relay guidance and upper-runtime confirmation responsibilities explicit. |

## Patterns To Avoid

- Source-level fingerprint patches, anti-detection claims, CAPTCHA avoidance,
  or proxy-based evasion.
- Auto-downloading a custom browser binary as the default path. OBU's core
  value is using the user's installed Chrome or Edge profile.
- Remote CDP exposure as a default interface. OBU should keep the local relay
  and MCP/CLI/SDK surfaces as the primary integration points.
- Marketing claims that imply bypassing site access controls.

## Four-Stage Application To OBU

1. **Comparison and scope**: This document defines the borrow/avoid boundary so
   future changes stay aligned with real-browser local automation.
2. **Diagnostics and observability**: `doctor` is the first troubleshooting
   surface, now including all-browser preflight and MCP structured output.
3. **Agent workflow**: Skill, MCP docs, SDK docs, and troubleshooting should all
   start from the same preflight, session, action, and cleanup contract.
4. **Product readiness**: Feature release notes, history, and quality score
   should describe the new user-visible reliability posture before release.

## Current Borrowed Outcomes

- `obu doctor --browser all --json` can inspect Chrome and Edge setup in one
  machine-readable preflight.
- MCP exposes the same `doctor` diagnostic object for MCP-first agents.
- The agent-facing docs route uncertain setup and failed `ping` through
  `doctor` before browser actions.
- Release and history docs record the diagnostic posture as a user-facing
  feature, not just an internal refactor.
