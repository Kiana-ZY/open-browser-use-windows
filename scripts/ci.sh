#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/scripts/runtime-tools.sh"

node_bin="$(find_node || true)"
if [[ -z "${node_bin}" ]]; then
  echo "node is required" >&2
  exit 127
fi

go_bin="$(find_go || true)"
if [[ -z "${go_bin}" ]]; then
  echo "go is required" >&2
  exit 127
fi

python_bin="${PYTHON:-}"
if [[ -z "${python_bin}" ]]; then
  if command -v python >/dev/null 2>&1; then
    python_bin="python"
  elif command -v python3 >/dev/null 2>&1; then
    python_bin="python3"
  else
    echo "python or python3 is required" >&2
    exit 127
  fi
fi

"${repo_root}/scripts/check-docs.sh"
"${repo_root}/scripts/check-repo-hygiene.sh"
"${repo_root}/scripts/check-action-pinning.sh"
"${node_bin}" "${repo_root}/scripts/generate-chrome-extension-icons.mjs"
"${repo_root}/scripts/package-chrome-extension.sh" >/dev/null
"${repo_root}/scripts/package-skill.sh" >/dev/null
(
  cd "${repo_root}"
  run_go "${go_bin}" test ./...
  with_tool_path "${node_bin}" pnpm -r --if-present test
)
(
  cd "${repo_root}/packages/open-browser-use-python"
  "${python_bin}" -m unittest
)
"${node_bin}" --check "${repo_root}/scripts/chrome-web-store-oauth.mjs"
"${node_bin}" --check "${repo_root}/scripts/generate-chrome-extension-icons.mjs"
"${node_bin}" --check "${repo_root}/scripts/package-chrome-extension-crx.mjs"
"${node_bin}" --check "${repo_root}/scripts/publish-chrome-web-store.mjs"
"${node_bin}" --check "${repo_root}/scripts/zip-tool.mjs"

while IFS= read -r file; do
  bash -n "$file"
done < <(find "${repo_root}/scripts" -type f -name '*.sh' | sort)

echo "基础 CI 检查通过"
