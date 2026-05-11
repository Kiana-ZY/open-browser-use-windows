#!/usr/bin/env bash

find_node() {
  local candidate

  if [[ -n "${NODE:-}" ]]; then
    candidate="${NODE}"
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
    if command -v "${candidate}" >/dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
  fi

  for candidate in \
    node \
    "/home/yuhua/.local/bin/node" \
    node.exe \
    "/mnt/c/Program Files/nodejs/node.exe" \
    "/c/Program Files/nodejs/node.exe"; do
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
    if command -v "${candidate}" >/dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
  done

  return 1
}

find_go() {
  local candidate

  if [[ -n "${GO:-}" ]]; then
    candidate="${GO}"
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
    if command -v "${candidate}" >/dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
  fi

  for candidate in \
    go \
    go.exe \
    "/mnt/c/Program Files/Go/bin/go.exe" \
    "/c/Program Files/Go/bin/go.exe"; do
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
    if command -v "${candidate}" >/dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
  done

  return 1
}

is_windows_exe() {
  [[ "$1" == *.exe ]]
}

windows_path() {
  local input_path="$1"

  if command -v wslpath >/dev/null 2>&1; then
    wslpath -w "${input_path}"
    return 0
  fi

  if command -v cygpath >/dev/null 2>&1; then
    cygpath -w "${input_path}"
    return 0
  fi

  printf '%s\n' "${input_path}"
}

with_tool_path() {
  local tool_bin="$1"
  shift

  PATH="$(dirname "${tool_bin}"):${PATH}" "$@"
}

go_arg_for_shell() {
  local go_bin="$1"
  local arg="$2"

  if is_windows_exe "${go_bin}" && [[ "${arg}" == /* ]]; then
    windows_path "${arg}"
    return
  fi

  printf '%s\n' "${arg}"
}

run_go() {
  local go_bin="$1"
  shift

  if is_windows_exe "${go_bin}" && command -v powershell.exe >/dev/null 2>&1; then
    local cwd_win
    local go_bin_win
    local command
    local quoted_args

    cwd_win="$(windows_path "$(pwd)")"
    go_bin_win="$(windows_path "${go_bin}")"
    quoted_args=""
    for arg in "$@"; do
      arg="$(go_arg_for_shell "${go_bin}" "${arg}")"
      arg="${arg//\'/\'\'}"
      quoted_args="${quoted_args} '${arg}'"
    done
    command="\$ErrorActionPreference = 'Stop'; Set-Location -LiteralPath '${cwd_win}'; & '${go_bin_win}'${quoted_args}; exit \$LASTEXITCODE"
    powershell.exe -NoProfile -NonInteractive -Command "${command}"
    return
  fi

  "${go_bin}" "$@"
}
