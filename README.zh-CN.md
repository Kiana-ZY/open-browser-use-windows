# Open Browser Use for Windows

Open Browser Use for Windows 是面向 Windows 的 Open Browser Use 版本。它通过
Chrome / Microsoft Edge 扩展、Go native host、CLI、SDK 和 MCP server，让 AI
Agent 操作用户真实浏览器 profile。

命令名保持不变：正式命令是 `open-browser-use`，简称仍然是 `obu`。

## 项目重点

| 模块 | Windows 支持 |
| --- | --- |
| 浏览器 | Chrome 和 Microsoft Edge |
| Native host | 由 Native Messaging 启动的 `open-browser-use.exe` |
| 客户端传输 | Windows 上使用 `127.0.0.1:19832` TCP relay |
| Agent 接入 | CLI、MCP、JS SDK、Python SDK、Go SDK 和 skill |
| 安装辅助 | `setup`、`setup beta`、`install-manifest` 支持 `--browser edge` |

## 架构

```text
Chrome / Edge MV3 extension
  -> Native Messaging stdio
  -> open-browser-use.exe
  -> Windows TCP 127.0.0.1:19832
  -> CLI / MCP / JS SDK / Python SDK / Go SDK
```

## 编译

```powershell
go build -o open-browser-use.exe ./cmd/open-browser-use
```

## 安装浏览器集成

Edge:

```powershell
.\open-browser-use.exe setup beta --browser edge
```

Chrome:

```powershell
.\open-browser-use.exe setup beta --browser chrome
```

命令会打开扩展管理页并显示扩展 ZIP。打开开发者模式后，把 ZIP 拖进扩展页面安装。

## 验证

```powershell
.\open-browser-use.exe version
.\open-browser-use.exe doctor
.\open-browser-use.exe doctor --browser all --json
.\open-browser-use.exe ping --json
.\open-browser-use.exe info --json
```

如果已经全局安装，可以用：

```powershell
obu version
obu doctor
obu doctor --browser all --json
obu ping --json
obu info --json
```

## 常用命令

每个任务使用一个唯一 session id。

```powershell
$env:OBU_SESSION_ID = "obu-task-$(Get-Date -Format yyyyMMddHHmmss)"
obu name-session --name "Task - OBU"
obu open-tab --url https://github.com/trending --json
obu page-info --max-chars 2000 --json
obu snapshot --limit 50 --json
obu finalize-tabs --keep "[]" --json
```

| 命令 | 说明 |
| --- | --- |
| `obu doctor --json` | 诊断 native host、manifest、relay 和扩展状态 |
| `obu doctor --browser all --json` | 一次性诊断 Chrome 和 Edge |
| `obu ping --json` | 连通性检查 |
| `obu info --json` | 扩展和 native host 元数据 |
| `obu user-tabs --json` | 列出全部浏览器标签页 |
| `obu open-tab --url URL --json` | 打开任务标签页 |
| `obu claim-tab --tab-id ID --json` | 接管已有用户标签页 |
| `obu page-info --max-chars N --json` | 获取标题、URL、readyState 和正文 |
| `obu snapshot --limit N --json` | 获取有限数量的可交互元素 |
| `obu click @1 --json` | 点击 snapshot ref |
| `obu fill @2 "text" --json` | 填写 snapshot ref |
| `obu screenshot --output file.png --json` | 截图 |
| `obu run -c "..."` | 执行轻量 action plan |
| `obu mcp` | 启动 stdio MCP server |

## SDK

JavaScript / TypeScript:

```bash
npm install open-browser-use-sdk
```

Python:

```bash
pip install open-browser-use-sdk
```

Go:

```bash
go get github.com/Kiana-ZY/open-browser-use-windows/packages/open-browser-use-go
```

## Agent Skill

可复用 skill 位于 `skills/open-browser-use/`，适合 Codex、Claude Code 和其它
shell-first Agent runtime。触发别名是 `@obu`，CLI 简称仍是 `obu`。

Codex / Claude Code 的 MCP 配置示例见 `docs/CODEX_AND_CLAUDE_USAGE.md`。

## 仓库结构

| 路径 | 作用 |
| --- | --- |
| `apps/chrome-extension/` | Chrome / Edge MV3 扩展 |
| `cmd/open-browser-use/` | Go CLI、native host 和 MCP server |
| `internal/host/` | Native host relay 和 Windows TCP 传输 |
| `internal/wire/` | Native Messaging frame codec |
| `packages/open-browser-use-cli/` | 暴露 `open-browser-use` 和 `obu` 的 npm CLI 包 |
| `packages/open-browser-use-js/` | JS/TS SDK |
| `packages/open-browser-use-python/` | Python SDK |
| `packages/open-browser-use-go/` | Go SDK |
| `skills/open-browser-use/` | Agent 使用指南 |
| `archive/` | 非核心研究资料、过程记录和本地快照 |

## Archive

`archive/` 保存不属于主运行路径、但仍有追溯价值的材料：

- `archive/process/`：执行计划、历史记录、旧模板文档和旧 HTML 指南。
- `archive/research/`：逆向资料、生成数据和研究用包。
- `archive/local-agent-snapshots/`：本地 agent skill 快照。

## 许可证

MIT。本项目基于 Open Browser Use 工作演进，整理为 Windows-first 的
Chrome / Edge 版本。
