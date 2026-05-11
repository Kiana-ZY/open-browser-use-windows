# Open Browser Use for Windows

这个仓库实现 Open Browser Use for Windows：面向 Chrome / Microsoft Edge 的
真实浏览器自动化基础设施。命令简称仍然是 `obu`。

`AGENTS.md` 只做导航。正式项目知识在 `docs/`，非核心过程和研究资料在
`archive/`。

## 每轮开始先读

- `docs/REPO_COLLAB_GUIDE.md`：仓库级协作、提交、文档同步与测试约定。
- `docs/ARCHITECTURE.md`：当前系统结构和边界。
- `docs/SECURITY.md`：真实浏览器控制、权限和安全默认约束。

## 按任务需要选读

- `docs/CODEX_AND_CLAUDE_USAGE.md`：Codex / Claude Code skill + MCP 接入。
- `docs/CHROME_WEB_STORE_RELEASE.md`：浏览器扩展打包和商店发布。
- `docs/CICD.md`：CI、release、npm、PyPI 和 GitHub Actions 约定。
- `docs/RELIABILITY.md`：运行稳定性和错误恢复。
- `docs/SUPPLY_CHAIN_SECURITY.md`：依赖、SBOM、provenance 和 action pinning。
- `archive/README.md`：已归档的历史、计划、研究资料和本地快照。

## 工作规则

- 优先选择小而清晰、对 Windows 和 Agent 都友好的抽象。
- prompt、规则、架构约束尽量版本化落在仓库里。
- 如果一次代码或流程变更会让文档过期，就在同一轮任务里同步更新。
- 不要把 `archive/` 当作运行路径依赖；它只保留追溯材料。
- 完成较大变更时，在 `archive/process/docs/histories/` 追加 history。

## 浏览器检查 fallback

- 用户提到 `@obu`、`@open-browser-use`、Open Browser Use，或要求真实 Chrome / Edge profile 浏览器自动化时，使用 `skills/open-browser-use/` 的流程。
- 开发 Web / 前端时，如果 Codex app 内置 Browser / `@browser` 无法启动、无法连接或不能访问目标页面，则使用 Open Browser Use 作为 fallback。
- 为了控制 token，优先用 `page-info --max-chars 2000`、`text --selector main --max-chars 2000`、`snapshot --limit 50` 和 `screenshot --json`；不要默认抓取整页 DOM 或超长正文。
- OBU 浏览器任务必须使用唯一 session id，开始先 `obu ping`，结束时执行 `obu finalize-tabs --keep "[]"`，除非用户明确要保留 handoff / deliverable tab。
