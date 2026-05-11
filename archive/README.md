# Archive

This directory stores material that remains useful for provenance, but is not
part of the Open Browser Use for Windows runtime path.

## Categories

- `process/`: historical execution plans, change histories, old template docs,
  product notes, and the previous generated documentation site.
- `research/`: reverse-engineering notes, generated metadata, reference
  snapshots, and research-only packages that informed the implementation.
- `local-agent-snapshots/`: local agent skill/config snapshots kept for
  traceability. These should not be treated as install sources.

## Rules

- Runtime code, release scripts, SDKs, and active docs should not import, copy,
  or require files from `archive/`.
- New process records should go under `archive/process/docs/histories/` or
  `archive/process/docs/exec-plans/`.
- Archived references may contain old repository names, version numbers, and
  compatibility identifiers by design.
