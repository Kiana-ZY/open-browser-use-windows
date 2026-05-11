"""Generate the Open Browser Use documentation HTML page (Chinese)."""
from pathlib import Path

CSS_VARS = """    --primary: #cc785c; --primary-hover: #a9583e;
    --ink: #141413; --body: #3d3d3a; --muted: #6c6a64;
    --hairline: #e6dfd8; --canvas: #faf9f5;
    --surface-soft: #f5f0e8; --surface-card: #efe9de;
    --surface-dark: #181715; --on-dark: #faf9f5;
    --on-dark-soft: #a09d96; --success: #5db872; --error: #c64545;"""

TEMPLATE = r"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Open Browser Use - Windows & Edge 完全指南</title>
<style>
  :root {__CSS_VARS__}
  *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
  body{font-family:"Inter","PingFang SC","Microsoft YaHei",-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:16px;line-height:1.6;color:var(--body);background:var(--canvas);-webkit-font-smoothing:antialiased}
  .hero{background:var(--surface-dark);color:var(--on-dark);padding:80px 24px 64px;text-align:center;border-bottom:1px solid rgba(255,255,255,.06)}
  .hero-badge{display:inline-block;background:var(--primary);color:#fff;font-size:13px;font-weight:500;padding:4px 14px;border-radius:100px;margin-bottom:24px;letter-spacing:.5px;text-transform:uppercase}
  .hero h1{font-family:"Georgia","PingFang SC","Microsoft YaHei",serif;font-size:clamp(34px,5vw,56px);font-weight:400;line-height:1.1;margin-bottom:16px}
  .hero .lead{font-size:18px;color:var(--on-dark-soft);max-width:540px;margin:0 auto 32px}
  .hero-cmds{display:flex;gap:10px;justify-content:center;flex-wrap:wrap}
  .hero-cmd{background:rgba(255,255,255,.06);border:1px solid rgba(255,255,255,.1);color:var(--on-dark);padding:8px 16px;border-radius:6px;font-family:"SF Mono","Fira Code","Consolas",monospace;font-size:14px}
  .container{max-width:760px;margin:0 auto;padding:0 24px}
  section{padding:56px 0}
  section+section{border-top:1px solid var(--hairline)}
  .section-label{font-size:12px;font-weight:600;text-transform:uppercase;letter-spacing:1px;color:var(--primary);margin-bottom:10px}
  h2{font-family:"Georgia","PingFang SC","Microsoft YaHei",serif;font-size:28px;font-weight:400;line-height:1.2;color:var(--ink);margin-bottom:20px}
  h3{font-size:18px;font-weight:600;color:#252523;margin:28px 0 10px}
  p{margin-bottom:14px}
  pre{background:var(--surface-soft);border:1px solid var(--hairline);border-radius:8px;padding:20px 24px;overflow-x:auto;font-family:"SF Mono","Fira Code","Consolas",monospace;font-size:14px;line-height:1.5;margin-bottom:20px;white-space:pre-wrap;word-break:break-word}
  code{font-family:"SF Mono","Fira Code","Consolas",monospace;font-size:13px;background:var(--surface-soft);padding:2px 6px;border-radius:4px;color:var(--ink)}
  pre code{background:none;padding:0;border-radius:0;color:inherit}
  table{width:100%;border-collapse:collapse;margin:20px 0}
  th{text-align:left;font-size:13px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.5px;padding:12px 16px;border-bottom:1px solid var(--hairline)}
  td{padding:12px 16px;border-bottom:1px solid var(--hairline);font-size:15px}
  td:first-child{font-family:"SF Mono","Fira Code","Consolas",monospace;font-size:13px;color:var(--ink)}
  .card{background:var(--surface-card);border:1px solid var(--hairline);border-radius:12px;padding:24px;margin:20px 0}
  .card h4{font-size:15px;font-weight:600;margin-bottom:8px;color:var(--ink)}
  .note{background:#fdf8f0;border-left:3px solid #e8a55a;padding:16px 20px;border-radius:0 8px 8px 0;margin:20px 0;font-size:14px;color:#6b5a3e}
  .footer{background:var(--surface-dark);color:var(--on-dark-soft);text-align:center;padding:40px 24px;font-size:14px}
  .footer a{color:var(--primary);text-decoration:none}
  .footer a:hover{text-decoration:underline}
</style>
</head>
<body>

<header class="hero">
  <div class="hero-badge">Windows &amp; Edge</div>
  <h1>Open Browser Use</h1>
  <p class="lead">将 AI 代理连接到你的 Microsoft Edge 浏览器。完整的浏览器自动化能力——标签页控制、CDP 命令、文件上传、剪贴板访问等。</p>
  <div class="hero-cmds">
    <span class="hero-cmd">obu info</span>
    <span class="hero-cmd">obu user-tabs</span>
    <span class="hero-cmd">obu open-tab --url</span>
    <span class="hero-cmd">obu cdp --method</span>
  </div>
</header>

<section>
  <div class="container">
    <div class="section-label">架构</div>
    <h2>工作原理</h2>
    <p>Open Browser Use 由三层组成，通过不同的传输层进行通信：</p>
    <pre>Edge / Chrome 扩展 (MV3)
    |  stdio（原生消息协议）
    v
原生主机中继 (open-browser-use.exe)
    |  TCP 127.0.0.1:19832（Windows）
    |  Unix 套接字（macOS / Linux）
    v
CLI / Python SDK / Go SDK</pre>
    <p>浏览器扩展控制标签页并执行 DevTools 命令。原生主机在扩展和 AI 代理之间中继 JSON-RPC 消息。所有通信均采用 4 字节长度前缀的 JSON 帧格式。</p>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">快速开始</div>
    <h2>安装</h2>

    <h3>1. 安装 CLI</h3>
    <p><code>open-browser-use</code> 命令也可简写为 <code>obu</code>。</p>
    <pre>npm install -g open-browser-use</pre>

    <h3>2. 注册到 Edge</h3>
    <p>安装原生消息主机清单，下载扩展 ZIP 包，并打开 Edge 扩展页面：</p>
    <pre>open-browser-use setup beta --browser edge</pre>

    <h3>3. 安装扩展</h3>
    <p>在 <code>edge://extensions/</code> 中开启<strong>开发者模式</strong>，然后将下载的 ZIP 文件拖入页面。扩展 ID 为 <code>pnbmoicbkopffjjgfgfglopechaiemkp</code>。</p>

    <h3>4. 验证</h3>
    <pre>open-browser-use info</pre>
    <p>成功响应包含版本和扩展元数据。</p>

    <div class="note">
      <strong>Edge 与 Chrome 的区别：</strong>在 <code>setup</code>、<code>setup beta</code> 和 <code>install-manifest</code> 命令中使用 <code>--browser edge</code>。其余所有 CLI 命令完全通用。macOS 仅支持 Chrome。
    </div>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">CLI 命令参考</div>
    <h2>命令</h2>
    <p>每个命令都接受 <code>--session-id</code> 参数。代理任务务必提供唯一的会话 ID。</p>

    <h3>查看类</h3>
    <table>
      <tr><th>命令</th><th>说明</th></tr>
      <tr><td>obu info</td><td>扩展版本和元数据</td></tr>
      <tr><td>obu ping</td><td>轻量级连通性检查</td></tr>
      <tr><td>obu tabs</td><td>列出会话标签页</td></tr>
      <tr><td>obu user-tabs</td><td>列出所有浏览器标签页</td></tr>
      <tr><td>obu history --query "..." --limit N</td><td>搜索浏览历史</td></tr>
    </table>

    <h3>标签页操作</h3>
    <table>
      <tr><th>命令</th><th>说明</th></tr>
      <tr><td>obu open-tab --url URL</td><td>打开新的会话标签页</td></tr>
      <tr><td>obu claim-tab --tab-id ID</td><td>认领已有的用户标签页</td></tr>
      <tr><td>obu navigate --tab-id ID --url URL</td><td>导航标签页</td></tr>
      <tr><td>obu name-session --name "..."</td><td>命名任务标签页分组</td></tr>
      <tr><td>obu finalize-tabs --keep JSON</td><td>关闭或移交会话标签页</td></tr>
    </table>

    <h3>高级</h3>
    <table>
      <tr><th>命令</th><th>说明</th></tr>
      <tr><td>obu cdp --tab-id ID --method M --params JSON</td><td>Chrome DevTools 协议</td></tr>
      <tr><td>obu move-mouse --tab-id ID --x PX --y PX</td><td>移动光标覆盖层</td></tr>
      <tr><td>obu wait-file-chooser --tab-id ID</td><td>等待文件选择器对话框</td></tr>
      <tr><td>obu set-file-chooser-files --file-chooser-id ID --file PATH</td><td>设置上传文件</td></tr>
      <tr><td>obu call --method M --params JSON</td><td>原始 JSON-RPC 调用</td></tr>
    </table>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">批量脚本</div>
    <h2>行动方案脚本</h2>
    <p>使用 <code>obu run</code> 进行多步骤工作流，无需编写 SDK 代码：</p>
    <pre>SET OBU_SESSION_ID=obu-demo-001
obu run --session-id %OBU_SESSION_ID% -c "name-session 调研 - OBU
open-tab --url https://github.com/trending
wait-load domcontentloaded
page-info
finalize-tabs []"</pre>
    <p>每个行动行共享同一个会话和轮次。<code>open-tab</code> 和 <code>claim-tab</code> 会为后续的标签页范围内的操作设置默认标签页。</p>

    <h3>支持的脚本动作</h3>
    <table>
      <tr><th>动作</th><th>说明</th></tr>
      <tr><td>ping</td><td>连接检查</td></tr>
      <tr><td>info</td><td>扩展元数据</td></tr>
      <tr><td>tabs</td><td>列出会话标签页</td></tr>
      <tr><td>user-tabs</td><td>列出所有浏览器标签页</td></tr>
      <tr><td>name-session "名称"</td><td>设置标签页分组名称</td></tr>
      <tr><td>open-tab</td><td>创建标签页（可选：--url）</td></tr>
      <tr><td>claim-tab ID</td><td>认领已有标签页</td></tr>
      <tr><td>navigate URL</td><td>导航标签页</td></tr>
      <tr><td>wait-load</td><td>等待页面加载</td></tr>
      <tr><td>page-info</td><td>获取标题、URL、文本内容</td></tr>
      <tr><td>cdp METHOD</td><td>CDP 命令（--params JSON）</td></tr>
      <tr><td>history</td><td>搜索历史（--query）</td></tr>
      <tr><td>move-mouse X Y</td><td>移动光标覆盖层</td></tr>
      <tr><td>wait-file-chooser</td><td>等待文件选择器</td></tr>
      <tr><td>set-file-chooser-files ID FILES</td><td>设置上传文件</td></tr>
      <tr><td>finalize-tabs</td><td>关闭/移交标签页（--keep JSON）</td></tr>
      <tr><td>turn-ended</td><td>结束当前轮次</td></tr>
      <tr><td>call METHOD</td><td>原始 JSON-RPC（--params JSON）</td></tr>
    </table>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">会话管理</div>
    <h2>标签页生命周期</h2>
    <p><strong>会话</strong>是一个代理任务所创建或认领的浏览器标签页的逻辑分组。标签页必须在每个轮次结束时进行收尾。</p>

    <div class="card">
      <h4>1. 开始会话</h4>
      <p>选取一个唯一的会话 ID：<code>obu-任务名-时间戳</code></p>
      <pre>SET OBU_SESSION_ID=obu-research-20260511</pre>
    </div>

    <div class="card">
      <h4>2. 命名标签页分组</h4>
      <pre>obu name-session --session-id %OBU_SESSION_ID% --name "调研 - OBU"</pre>
    </div>

    <div class="card">
      <h4>3. 打开或认领标签页</h4>
      <pre>obu open-tab --session-id %OBU_SESSION_ID% --url https://example.com
obu claim-tab --session-id %OBU_SESSION_ID% --tab-id 123456</pre>
    </div>

    <div class="card">
      <h4>4. 收尾标签页</h4>
      <table>
        <tr><th>状态</th><th>行为</th></tr>
        <tr><td>deliverable</td><td>将标签页移至用户可查看的分组中</td></tr>
        <tr><td>handoff</td><td>将标签页保留在会话分组中以便后续继续工作</td></tr>
      </table>
      <pre style="margin-top:12px">REM 关闭所有会话标签页（默认行为）
obu finalize-tabs --session-id %OBU_SESSION_ID% --keep "[]"

REM 保留一个交付标签页
obu finalize-tabs --session-id %OBU_SESSION_ID% --keep "[{\"tabId\":123,\"status\":\"deliverable\"}]"

REM 移交待后续处理
obu finalize-tabs --session-id %OBU_SESSION_ID% --keep "[{\"tabId\":456,\"status\":\"handoff\"}]"</pre>
    </div>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">SDK 集成</div>
    <h2>Python 与 Go 集成</h2>

    <h3>Python</h3>
    <pre>from open_browser_use import OpenBrowserUseClient

client = OpenBrowserUseClient(socket_path="")
browser = client.connect()
tab = browser.new_tab("https://example.com")
print(tab.title())
print(tab.dom_snapshot())
tab.close()
browser.close()</pre>
    <p>在 Windows 上 <code>socket_path</code> 参数会被忽略。客户端自动连接到 <code>127.0.0.1:19832</code>。</p>

    <h3>Go</h3>
    <pre>import obu "github.com/ifuryst/open-codex-browser-use/packages/open-browser-use-go"

browser, _ := obu.ConnectActive(obu.Options{})
defer browser.Close()
result, _ := browser.Client.GetInfo()
tab := browser.NewTab("https://example.com", obu.LoadStateLoad, obu.DefaultNavigationTimeout)</pre>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">MCP 服务器</div>
    <h2>模型上下文协议</h2>
    <p>对于支持本地 MCP 服务器的 AI 运行时环境：</p>
    <pre>[mcp_servers.open_browser_use]
command = "obu"
args = ["mcp", "--session-id", "obu-task-001"]</pre>
    <p>暴露的工具：</p>
    <table>
      <tr><th>工具</th><th>说明</th></tr>
      <tr><td>user_tabs</td><td>列出所有浏览器标签页</td></tr>
      <tr><td>open_tab</td><td>打开新标签页</td></tr>
      <tr><td>claim_tab</td><td>认领已有标签页</td></tr>
      <tr><td>navigate</td><td>导航标签页到指定 URL</td></tr>
      <tr><td>wait_load</td><td>等待页面加载状态</td></tr>
      <tr><td>page_info</td><td>获取页面标题、URL、文本</td></tr>
      <tr><td>cdp</td><td>执行 CDP 命令</td></tr>
      <tr><td>history</td><td>搜索浏览历史</td></tr>
      <tr><td>run_action_plan</td><td>执行批量行动方案</td></tr>
      <tr><td>finalize_tabs</td><td>关闭或移交会话标签页</td></tr>
      <tr><td>call</td><td>不受限的 JSON-RPC 调用</td></tr>
    </table>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">故障排除</div>
    <h2>常见问题</h2>
    <table>
      <tr><th>症状</th><th>解决方案</th></tr>
      <tr><td>原生消息主机未找到</td><td>缺少注册表键。重新运行 <code>open-browser-use setup beta --browser edge</code> 或手动创建注册表键。</td></tr>
      <tr><td>active socket 注册表不可用</td><td>主机未运行。重启 Edge 或确保扩展已启用。</td></tr>
      <tr><td>连接被拒绝（TCP 19832）</td><td>主机进程已退出。重启 Edge 以重新启动。</td></tr>
      <tr><td>命令超时</td><td>扩展可能处于空闲状态。点击 Edge 工具栏中的扩展图标。</td></tr>
      <tr><td>扩展未加载</td><td>需要在 <code>edge://extensions/</code> 中开启开发者模式。</td></tr>
    </table>

    <h3>快速修复</h3>
    <pre>open-browser-use install-manifest --browser edge</pre>

    <h3>平台差异</h3>
    <table>
      <tr><th>项目</th><th>Windows</th><th>macOS / Linux</th></tr>
      <tr><td>中继传输</td><td>TCP 127.0.0.1:19832</td><td>Unix 套接字</td></tr>
      <tr><td>套接字注册表</td><td>%TEMP%\open-browser-use\active.json</td><td>/tmp/open-browser-use/active.json</td></tr>
      <tr><td>清单位置</td><td>%LOCALAPPDATA%\...\NativeMessagingHosts\</td><td>~/Library/... 或 ~/.config/...</td></tr>
      <tr><td>Edge 支持</td><td>--browser edge</td><td>不支持</td></tr>
    </table>
  </div>
</section>

<section>
  <div class="container">
    <div class="section-label">安全规范</div>
    <h2>操作规范</h2>
    <ul style="list-style:none;padding:0">
      <li style="padding:8px 0;border-bottom:1px solid var(--hairline)">将浏览器视为用户的真实数据。绝不检查 Cookie、密码或会话存储。</li>
      <li style="padding:8px 0;border-bottom:1px solid var(--hairline)">安装、启用、上传、提交、购买或删除操作前先征得用户同意。</li>
      <li style="padding:8px 0;border-bottom:1px solid var(--hairline)">绝不要猜测标签页 ID——始终先列出再使用返回的 ID。</li>
      <li style="padding:8px 0;border-bottom:1px solid var(--hairline)">每个任务一个会话 ID。不要复用 <code>obu-cli</code>。</li>
      <li style="padding:8px 0;border-bottom:1px solid var(--hairline)">每个轮次结束时收尾标签页。默认不保留任何标签页。</li>
      <li style="padding:8px 0">只有在没有更安全的便捷命令时才使用 <code>call --method</code>。</li>
    </ul>
  </div>
</section>

<footer class="footer">
  <p>Open Browser Use &middot; <a href="https://github.com/iFurySt/open-codex-browser-use">GitHub</a></p>
  <p style="margin-top:8px">Windows &amp; Edge 适配版本</p>
</footer>

</body>
</html>"""

def build():
    html = TEMPLATE.replace("__CSS_VARS__", CSS_VARS)
    out = Path("doc/index.html")
    out.write_text(html, encoding="utf-8")
    print(f"HTML written: {out.stat().st_size} bytes")

if __name__ == "__main__":
    build()
