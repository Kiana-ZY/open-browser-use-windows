# 20260511-1324 codex-claude-usage-and-mcp-contract

## 用户诉求

补齐两件事：

- 说明在 Codex 和 Claude Code 里应该如何使用 Open Browser Use。
- 把 agent 高频路径继续收口成更稳定的一套对象协议。

## 本次改动

- 新增 `docs/CODEX_AND_CLAUDE_USAGE.md`，单独说明 Codex / Claude Code 的
  skill 放置方式、MCP 配置、推荐工作流、提示词示例和使用边界。
- 在 `cmd/open-browser-use/mcp.go` 里把 MCP 常用工具的
  `structuredContent` 对齐到和 CLI `--json` 一致的稳定对象形状。
- 在 `cmd/open-browser-use/mcp_test.go` 里补充归一化测试，覆盖
  `user_tabs`、`ping`、`page_info`。
- 在 `README.md`、`packages/open-browser-use-cli/README.md` 和
  `skills/open-browser-use/references/sdk-and-protocol.md` 补上新文档入口，
  并明确 MCP 也遵循相同的 agent-facing contract。

## 设计动机

前一轮已经把直连 CLI `--json` 收成更稳定的对象协议，但如果 MCP 继续把底层
 Browser RPC 结果原样透出，agent 仍然要为 CLI 和 MCP 各写一套适配分支。

这次继续在 MCP 入口做薄归一化，让常用路径尽量满足：

- 同一个动作在 CLI / MCP 拿到同一语义形状
- skill、CLI 文档、MCP 文档说的是同一套 contract
- agent 不必再区分“裸数组 / result 包装 / 最终对象”三种层级

## 关键文件

- `docs/CODEX_AND_CLAUDE_USAGE.md`
- `cmd/open-browser-use/mcp.go`
- `cmd/open-browser-use/mcp_test.go`
- `packages/open-browser-use-cli/README.md`
- `skills/open-browser-use/references/sdk-and-protocol.md`
- `README.md`

## 验证

已运行：

```text
go test ./cmd/open-browser-use
```
