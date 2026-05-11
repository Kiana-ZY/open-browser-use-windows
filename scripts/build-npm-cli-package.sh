#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_dir="${repo_root}/packages/open-browser-use-cli"
native_dir="${package_dir}/native"

source "${repo_root}/scripts/runtime-tools.sh"

go_bin="$(find_go || true)"
if [[ -z "${go_bin}" ]]; then
  echo "go is required to build the npm CLI package" >&2
  exit 127
fi

build_cli() {
  local goos="$1"
  local goarch="$2"
  local output_path="$3"

  if is_windows_exe "${go_bin}" && command -v wslpath >/dev/null 2>&1; then
    local repo_root_win
    local output_path_win
    local go_bin_win
    repo_root_win="$(wslpath -w "${repo_root}")"
    output_path_win="$(wslpath -w "${output_path}")"
    go_bin_win="$(wslpath -w "${go_bin}")"
    powershell.exe -NoProfile -Command "\$ErrorActionPreference = 'Stop'; \$env:CGO_ENABLED = '0'; \$env:GOOS = '${goos}'; \$env:GOARCH = '${goarch}'; Set-Location -LiteralPath '${repo_root_win}'; & '${go_bin_win}' build -trimpath -ldflags='-s -w' -o '${output_path_win}' './cmd/open-browser-use'; exit \$LASTEXITCODE"
    return
  fi

  (
    cd "${repo_root}"
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
      "${go_bin}" build -trimpath -ldflags="-s -w" -o "${output_path}" ./cmd/open-browser-use
  )
}

rm -rf "${native_dir}"
mkdir -p "${native_dir}"

targets=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  output_dir="${native_dir}/${goos}-${goarch}"
  output_name="open-browser-use"
  if [ "${goos}" = "windows" ]; then
    output_name="${output_name}.exe"
  fi
  mkdir -p "${output_dir}"
  build_cli "${goos}" "${goarch}" "${output_dir}/${output_name}"
done
