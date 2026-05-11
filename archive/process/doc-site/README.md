---
name: Open Browser Use — Windows & Edge 指南
description: 在 Windows 上安装、配置和使用 Open Browser Use 配合 Microsoft Edge 的完整指南。涵盖 CLI 命令、会话管理、标签页生命周期、SDK 使用和故障排除。
---

# Open Browser Use

## Windows & Edge 完全指南

Open Browser Use 将 MV3 浏览器扩展、本地原生消息中继、CLI 和多语言 SDK 连接在一起，让 AI 代理能够自动化操作真实浏览器。原本为 Chrome 构建，现已完整适配 **Windows 上的 Microsoft Edge**。

---

## 架构

```
┌─────────────┐     stdio      ┌──────────────┐     TCP :19832     ┌─────────────┐
│  Edge / Chrome │ ◄────────── │  原生主机     │ ◄─────────────── │  CLI / SDK  │
│  扩展         │ ──────────► │  (中继)       │ ────────────────► │  客户端     │
└─────────────┘               └──────────────┘                   └─────────────┘
  MV3 扩展                     open-browser-use.exe               obu / Python / Go
```

- 浏览器扩展通过 **stdin/stdout**（Chrome 原生消息协议）与原生主机通信。
- 原生主机通过 **TCP localhost:19832**（Windows）或 Unix 域套接字（macOS/Linux）将消息中继给 SDK 客户端。
- 所有传输层使用相同的 JSON-RPC 帧格式。

---

## 快速开始

### 1. 安装 CLI

`open-browser-use` 命令全局可用，简写为 `obu`。

### 2. 注册到 Edge

```sh
open-browser-use setup beta --browser edge
```

这会将原生消息主机清单安装到：

```
%LOCALAPPDATA%\Microsoft\Edge\User Data\NativeMessagingHosts\
```

下载扩展 ZIP 包，打开 `edge://extensions/`，并在资源管理器中显示 ZIP 文件。

### 3. 安装扩展

在 `edge://extensions/` 中开启**开发者模式**，然后将 ZIP 文件拖入页面。

### 4. 验证

```sh
open-browser-use info
```

成功的响应会包含扩展版本和元数据。

---

## CLI 命令参考

每个命令都接受 `--session-id` 参数来限定操作范围。代理任务务必提供唯一的会话 ID。

### 查看类

| 命令 | 说明 |
|---------|-------------|
| `obu info` | 扩展版本和元数据 |
| `obu ping` | 轻量级连通性检查 |
| `obu tabs` | 列出会话标签页 |
| `obu user-tabs` | 列出所有浏览器标签页 |
| `obu history --query "…" --limit 20` | 搜索浏览历史 |

### 标签页操作

| 命令 | 说明 |
|---------|-------------|
| `obu open-tab --url https://…` | 打开新的会话标签页 |
| `obu claim-tab --tab-id <id>` | 认领已有的浏览器标签页 |
| `obu navigate --tab-id <id> --url https://…` | 导航标签页 |
| `obu name-session --name "任务名 - OBU"` | 命名标签页分组 |

### 高级

| 命令 | 说明 |
|---------|-------------|
| `obu cdp --tab-id <id> --method <method> --params '<json>'` | Chrome DevTools 协议 |
| `obu move-mouse --tab-id <id> --x <px> --y <px>` | 移动光标覆盖层 |
| `obu wait-file-chooser --tab-id <id>` | 等待文件选择器 |
| `obu set-file-chooser-files --file-chooser-id <id> --file <path>` | 设置上传文件 |
| `obu wait-load --tab-id <id> --state domcontentloaded` | 等待页面加载 |
| `obu page-info --tab-id <id>` | 获取标题、URL、文本内容 |
| `obu call --method <method> --params '<json>'` | 原始 JSON-RPC 调用 |

### 批量脚本

```sh
obu run --session-id "%OBU_SESSION_ID%" -c "
name-session 调研 - OBU
open-tab --url https://github.com/trending
wait-load domcontentloaded
page-info
finalize-tabs []
"
```

---

## 会话与标签页生命周期

**会话**是一个代理任务创建或认领的浏览器标签页的逻辑分组。

1. **开始** — 创建唯一会话 ID：`obu-<任务>-<时间戳>`。
2. **命名** — 在打开标签页前执行 `obu name-session --name "任务名 - OBU"`。
3. **打开或认领** — `open-tab` 创建新页面，`claim-tab` 复用已有页面。
4. **操作** — 导航、执行 JS、通过 CDP 截图。
5. **收尾** — 关闭或移交标签页：

```sh
# 关闭所有会话标签页（默认）
obu finalize-tabs --keep "[]"

# 保留一个交付标签页
obu finalize-tabs --keep "[{\"tabId\":123,\"status\":\"deliverable\"}]"

# 移交待后续处理
obu finalize-tabs --keep "[{\"tabId\":456,\"status\":\"handoff\"}]"
```

| 状态 | 行为 |
|--------|----------|
| `deliverable` | 将标签页移至 `✅ Open Browser Use` 分组 — 面向用户的输出 |
| `handoff` | 将标签页保留在会话分组中 — 任务仍在进行 |

---

## SDK 使用

### Python

```python
from open_browser_use import OpenBrowserUseClient

client = OpenBrowserUseClient(socket_path="")
browser = client.connect()

tab = browser.new_tab("https://example.com")
print(tab.title())
print(tab.dom_snapshot())
tab.close()
browser.close()
```

在 Windows 上 `socket_path` 参数会被忽略——客户端自动连接到 `127.0.0.1:19832`。

### Go

```go
import obu "github.com/ifuryst/open-codex-browser-use/packages/open-browser-use-go"

browser, _ := obu.ConnectActive(obu.Options{})
defer browser.Close()

result, _ := browser.Client.GetInfo()
// 提供了 Browser、Cdp、Tab 包装器
tab := browser.NewTab("https://example.com", obu.LoadStateLoad, obu.DefaultNavigationTimeout)
```

---

## MCP 服务器

对于支持本地 MCP 服务器（基于 stdio）的运行时环境：

```toml
[mcp_servers.open_browser_use]
command = "obu"
args = ["mcp", "--session-id", "obu-<任务ID>"]
```

暴露的工具包括 `user_tabs`、`open_tab`、`claim_tab`、`navigate`、`wait_load`、`page_info`、`cdp`、`history`、`run_action_plan`、`finalize_tabs` 和不受限的 `call`。

---

## 故障排除

| 症状 | 检查 |
|---------|-------|
| `Specified native messaging host not found` | 验证注册表键：`HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.ifuryst.open_browser_use.extension` |
| `active socket registry is unavailable` | 主机进程未运行。确认 Edge 已打开且扩展已启用。 |
| `Connection refused`（TCP 19832） | 主机进程已退出或未启动。重启 Edge。 |
| 命令超时 | 扩展可能处于空闲状态。在 Edge 中与扩展交互。 |

### 快速修复

```sh
# 重新注册原生主机
open-browser-use install-manifest --browser edge

# 重新创建注册表键
powershell -Command "New-Item -Path 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.ifuryst.open_browser_use.extension' -Force"
```

---

## 平台差异

| 项目 | Windows | macOS / Linux |
|--------|---------|---------------|
| 中继传输 | TCP `127.0.0.1:19832` | Unix 套接字 |
| 原生主机清单 | `%LOCALAPPDATA%\…\NativeMessagingHosts\` | `~/Library/…` 或 `~/.config/…` |
| 套接字注册表 | `%TEMP%\open-browser-use\active.json` | `/tmp/open-browser-use/active.json` |
| Edge 支持 | `--browser edge` | 不支持 |

---

## 操作规范

- 将浏览器视为用户的真实数据。绝不检查 Cookie、密码或会话存储。
- 安装、启用、上传、提交、购买或删除操作前先征得用户同意。
- 绝不要猜测标签页 ID —— 始终先列后取。
- 每个任务一个会话 ID。不要复用 `obu-cli`。
- 每个轮次结束时收尾标签页。默认不保留任何标签页。
