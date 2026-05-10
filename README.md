# Open Browser Use — Windows & Edge

[原项目](https://github.com/iFurySt/open-codex-browser-use) 面向 macOS/Linux + Chrome。本项目进行了完整的 **Windows 适配**，新增 **Microsoft Edge** 支持。

[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## 改动

| 模块 | 改动 |
|------|------|
| **中继传输** | Unix 域套接字 → TCP `127.0.0.1:19832` |
| **Edge 支持** | `setup` / `setup beta` / `install-manifest` 增加 `--browser edge` |
| **注册表** | setup 时自动写入 Edge 注册表键 |
| **连接发现** | Windows 跳过 socket 文件扫描，直连 TCP |
| **Python SDK** | Named Pipe → TCP socket |
| **错误信息** | 修正 named-pipe 等误导性提示 |

## 架构

```
Edge / Chrome 扩展 (MV3)
    │  stdio (Native Messaging)
    ▼
Native Host 中继 (open-browser-use.exe)
    │  TCP 127.0.0.1:19832 (Windows)
    │  Unix Socket (macOS / Linux)
    ▼
CLI / Python SDK / Go SDK
```

## 快速开始

### 编译

```bash
go build -o open-browser-use.exe ./cmd/open-browser-use/
```

### 安装到 Edge

```bash
.\open-browser-use.exe setup beta --browser edge
```

在 `edge://extensions/` 开启**开发者模式**，将下载的 ZIP 拖入页面安装扩展。

### 验证

```bash
obu info
```

## CLI 命令

所有命令需带 `--session-id`，每个任务用一个唯一 ID。

| 命令 | 说明 |
|------|------|
| `obu info` | 扩展版本和元数据 |
| `obu ping` | 连通性检查 |
| `obu user-tabs` | 列出所有标签页 |
| `obu tabs` | 列出会话标签页 |
| `obu open-tab --url URL --session-id ID` | 打开新标签页 |
| `obu claim-tab --tab-id ID --session-id ID` | 认领已有标签页 |
| `obu navigate --tab-id ID --url URL --session-id ID` | 导航 |
| `obu cdp --tab-id ID --method M --params JSON --session-id ID` | CDP 命令 |
| `obu history --query "..." --limit N --session-id ID` | 搜索历史 |
| `obu finalize-tabs --keep JSON --session-id ID` | 关闭/移交会话标签页 |

```cmd
SET OBU_SESSION_ID=obu-task-20260511
obu open-tab --session-id %OBU_SESSION_ID% --url https://github.com/trending
obu cdp --session-id %OBU_SESSION_ID% --tab-id <id> --method Runtime.evaluate --params "{\"expression\":\"document.title\"}"
obu finalize-tabs --session-id %OBU_SESSION_ID% --keep "[]"
```

## SDK

### Python

```python
from open_browser_use import OpenBrowserUseClient

client = OpenBrowserUseClient(socket_path="")  # Windows 上忽略
browser = client.connect()
tab = browser.new_tab("https://example.com")
print(tab.title())
browser.close()
```

### Go

```go
import obu "github.com/ifuryst/open-codex-browser-use/packages/open-browser-use-go"

browser, _ := obu.ConnectActive(obu.Options{})
defer browser.Close()
result, _ := browser.Client.GetInfo()
```

## AI Agent Skill

Skill 文件在 `skills/open-browser-use/`，支持 Claude Code 和 Codex：

```
.claude/skills/open-browser-use/     # Claude Code
~/.codex/skills/open-browser-use/    # Codex
```

## 故障排除

| 症状 | 处理 |
|------|------|
| `TCP relay not available` | 重启 Edge |
| `Specified native messaging host not found` | `obu install-manifest --browser edge` |
| 命令超时 | 点击 Edge 工具栏中的扩展图标 |
| 端口 19832 连接被拒 | 主机进程已退出，重启 Edge |

## 许可证

基于 [iFurySt/open-codex-browser-use](https://github.com/iFurySt/open-codex-browser-use) 修改，Apache 2.0。
