#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

required_files=(
  "AGENTS.md"
  "README.md"
  "CONTRIBUTING.md"
  "archive/README.md"
  "docs/REPO_COLLAB_GUIDE.md"
  "docs/ARCHITECTURE.md"
  "docs/CHROME_WEB_STORE_LISTING.md"
  "docs/CHROME_WEB_STORE_RELEASE.md"
  "docs/CICD.md"
  "docs/CODEX_AND_CLAUDE_USAGE.md"
  "docs/PRIVACY_POLICY.md"
  "docs/RELIABILITY.md"
  "docs/SECURITY.md"
  "docs/SUPPLY_CHAIN_SECURITY.md"
  "docs/releases/feature-release-notes.md"
  "archive/process/docs/exec-plans/templates/execution-plan.md"
  "archive/process/docs/exec-plans/tech-debt-tracker.md"
  "archive/process/docs/histories/template.md"
)

missing=0

for path in "${required_files[@]}"; do
  if [[ ! -f "${repo_root}/${path}" ]]; then
    echo "缺少必要文件: ${path}"
    missing=1
  fi
done

for dir in archive/process/docs/exec-plans/active archive/process/docs/exec-plans/completed archive/process/docs/histories; do
  if [[ ! -d "${repo_root}/${dir}" ]]; then
    echo "缺少必要目录: ${dir}"
    missing=1
  fi
done

if ! grep -q "docs/" "${repo_root}/AGENTS.md"; then
  echo "AGENTS.md 应明确指向 docs/，说明它是仓库知识的正式来源"
  missing=1
fi

if ! grep -q "archive/" "${repo_root}/AGENTS.md"; then
  echo "AGENTS.md 应明确指向 archive/，说明归档资料的位置"
  missing=1
fi

if [[ "${missing}" -ne 0 ]]; then
  exit 1
fi

echo "文档骨架检查通过"
