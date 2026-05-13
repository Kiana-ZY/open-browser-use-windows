# 稳定性与可运维性

这里用来定义项目的运行质量底线。

## 本地健康检查

`open-browser-use doctor` 是安装和连接问题的第一排查入口。它会检查：

- CLI 版本、操作系统和架构。
- 选中的浏览器，默认 Chrome，可用 `--browser edge` 检查 Edge。
- stable native host 路径是否存在。
- Chrome / Edge native messaging manifest 是否存在、host name 是否匹配、
  manifest `path` 是否指向 stable native host、`allowed_origins` 是否包含扩展来源。
- Windows 上的 native messaging registry key 是否存在。
- relay 是否能 `ping`，以及 extension `getInfo` 是否能返回版本和 extension id。
- 已安装但未连接、版本落后、manifest 缺失等状态的下一步建议。

常用命令：

```sh
open-browser-use doctor
open-browser-use doctor --browser edge
open-browser-use doctor --browser all --json
open-browser-use doctor --json
```

`doctor --json` 输出结构化报告，适合 agent runtime、CI smoke check 和 issue
模板收集诊断信息。报告里包含 `ok`、`socket`、`nativeHost`、
`browserExtension`、`checks` 和 `nextSteps` 字段；分享诊断结果前仍应按
`docs/SECURITY.md` 要求检查本地路径和个人信息。

`doctor --browser all --json` 会返回聚合报告：

```json
{
  "ok": false,
  "browsers": [
    { "browser": "chrome", "ok": false },
    { "browser": "edge", "ok": false }
  ],
  "nextSteps": []
}
```

Agent runtime 和 issue 模板应优先用这个 all-browser preflight 收集环境状态，
再根据用户实际要控制 Chrome 还是 Edge 选择后续操作。

## 本地真实浏览器 smoke

`scripts/smoke-obu-example.sh` 会用唯一 session 打开 `https://example.com`，
执行 `page-info`、`snapshot`、`screenshot` 和 `finalize-tabs`，并把 trace 与截图写到
`_local_nonessential/`：

```sh
bash ./scripts/smoke-obu-example.sh
```

这条 smoke 适合在发布前或修浏览器链路后跑一次。它依赖当前用户已经安装并连接
Chrome / Edge 扩展和 native host；如果失败，先跑
`open-browser-use doctor --browser all --json` 判断是 manifest、relay 还是扩展连接问题。

## 维护内容

建议继续维护的内容包括：

- 启动、健康检查和基本可用性要求。
- 日志、指标、链路的采集和访问约定。
- timeout、retry、backoff 的默认策略。
- 本地和 CI 的关键路径验证方式。
- 常见故障、排查路径和恢复步骤。

CI/CD 流程结构和 release 自动化的默认方案，统一写在 `docs/CICD.md`。
