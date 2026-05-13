#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="_local_nonessential"
mkdir -p "$ARTIFACT_DIR"

GO_BIN="${GO_BIN:-}"
if [[ -z "$GO_BIN" ]]; then
  if command -v go >/dev/null 2>&1; then
    GO_BIN="$(command -v go)"
  elif command -v go.exe >/dev/null 2>&1; then
    GO_BIN="$(command -v go.exe)"
  elif command -v powershell.exe >/dev/null 2>&1; then
    GO_BIN="$(powershell.exe -NoProfile -Command '(Get-Command go -ErrorAction SilentlyContinue).Source' | tr -d '\r')"
  fi
fi

if [[ -z "$GO_BIN" ]]; then
  echo "go executable not found; set GO_BIN to the Go binary path" >&2
  exit 1
fi

SESSION_ID="${OBU_SESSION_ID:-obu-smoke-$(date +%Y%m%d%H%M%S)}"
TRACE_LOG="$ARTIFACT_DIR/smoke-obu-example-trace.jsonl"
SCREENSHOT="$ARTIFACT_DIR/smoke-obu-example.png"

rm -f "$TRACE_LOG" "$SCREENSHOT"

"$GO_BIN" run ./cmd/open-browser-use run \
  --session-id "$SESSION_ID" \
  --trace-log "$TRACE_LOG" \
  -c '
name-session "Smoke - OBU"
open-tab https://example.com
wait-load domcontentloaded
page-info --max-chars 1200
snapshot --limit 20
screenshot --output _local_nonessential/smoke-obu-example.png
finalize-tabs []
'

if [[ ! -s "$SCREENSHOT" ]]; then
  echo "expected non-empty screenshot at $SCREENSHOT" >&2
  exit 1
fi

if [[ ! -s "$TRACE_LOG" ]]; then
  echo "expected non-empty trace log at $TRACE_LOG" >&2
  exit 1
fi

echo "OBU example smoke passed"
echo "session: $SESSION_ID"
echo "trace: $TRACE_LOG"
echo "screenshot: $SCREENSHOT"
