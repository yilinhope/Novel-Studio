# M6 Progress Log

## Session: 2026-09-24 — M6 kickoff

- 读取用户提供的 M6 完整要求。
- 读取 `planning-with-files`、GitNexus exploring/impact、executing-plans、TDD、verification-before-completion 技能。
- 拉取远端 `main` 后确认 M5 PR #5 已合并：`origin/main` 为 `72add9b`。
- 从 `origin/main` 创建 `codex/m6-v1`；创建时工作区干净。
- 重置本阶段计划文件，后续所有审计发现、错误和验证结果持续写入本文件与 `findings.md`。
- 完成第一轮 GitNexus 图审计：`Host.New` 候选歧义但 Host 版本风险 LOW；`StartPrepared`、`CoCreateStream`、`ImportFrom`、`ConfigureModels` 直接调用方风险 LOW；Host `Snapshot` 的候选结果包含 UNKNOWN，已记录为需要源码补证而非安全结论。
- 完成 Core 初审：Quick Start、Co-create、Import checkpoint、TXT/EPUB Export、Provider 配置合并、Usage/Budget 持久化与 Host 生命周期边界已定位；Start from Outline 仍需确认是否存在正式公开入口。
- 补充源码审计文档 `docs/studio-m6-audit.md` 并提交 `84a942c`；审计确认没有独立 outline parser，GUI 复用 `/start` 的文件 prompt 语义。
- 修正 Studio Host 创建时的配置根目录：`newHostForProject` 使用 `EffectiveConfigPathFromDir(ProjectRoot)`，CLI 未传 option 时仍保持原 cwd 语义；补项目根配置回归测试。
- 新增 M6 Go Facade/Bridge：Quick Start、Outline preview、冷/阶段 Co-create、Core Import status/recovery、TXT/EPUB Export、Provider/Model/Thinking、Budget、Usage；只读读取不创建 Host，显式写操作沿用现有 Host/lifecycle/exclusive。
- 共创恢复仅读取既有 `meta/sessions/cocreate.jsonl`，不新增持久化格式；项目切换在 Create/Import/Co-create 长操作期间受 operation 与 Engine switch gate 保护。
- 新增前端 `createProjectStore`、`importStore`、`configStore`、`exportStore` 与对应 Create/Import/Settings/Export 页面；事件与异步响应按 projectId/generation/request sequence 过滤，API key 仅显示 Core 脱敏结果。
- 设置页补齐 Core 已有的 Agent 角色模型与 Reasoning Effort 控制，仍通过既有 `SwitchModel` / `SetRoleThinking` 写回并重新读取 effective config。
- 最终 GitNexus 检查发现跨 Wails 动态边界带来的 `UNKNOWN`/`critical` 风险包络，已对 `Host.New` 做 HIGH 影响说明，并以调用点搜索、全量测试、vet 与 Wails 构建补证，未发现 CLI 默认路径或现有 Core 语义变化。
- M6 冻结修正：Budget 改为项目层补丁写入；Quick Start/Co-create 增加 requestId 与 ack 前 terminal event 暂存；共创恢复 hydrate 已落盘历史/草稿并可显式 Resume；Import 文案改为 UTF-8 / GB18030 文本文件。

## Verification

| Command | Result |
|---|---|
| `git fetch origin main codex/m5-review-steer-bc` | passed |
| `git switch -c codex/m6-v1 origin/main` | passed |
| `go test ./... -count=1` | passed: 1017 tests |
| `go vet ./...` | passed |
| `npm test -- --run` | passed: 37 tests |
| `npm run build` | passed |
| `go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 build` | passed: Windows production `cmd/novel-studio/build/bin/Novel-Studio.exe` |

## Session: 2026-09-24 — M7 kickoff

- 读取用户提供的 M7 Regression / Hardening / Release Readiness 要求；确认 M7 不新增产品功能、不扩展 V2、不重写 Core。
- 确认当前 M6 冻结基线为 `5de0506`，工作区在 M7 审计开始前干净，PR #6 CI 4/4 通过。
- 已创建 M7 计划阶段：源码审计 → CLI/GUI 回归与生命周期 → 跨进程写入/项目切换 → 长篇/Windows/发布验证。
- 首轮源码审计进行中；下一步先输出 Studio API、路径、Host 创建、异步 identity、事实源和 secret 扫描结果，再对真实回归直接实施最小修复。

## Session: 2026-09-24 — M7 hardening

- Import ack 前事件丢失已修复：ImportOptions/ImportStatus/OperationAck 共享 requestId，前端在 ack 前缓存同 requestId 的事件，ack 后回放；新增前端回归测试。
- SaveChapter/Budget 写入复用 Core book lease；无 Host 时临时取得同一 `.ainovel.lock`，不创建 Engine Session；新增已有 lease 拒绝 Save 的 Go 回归测试。
- ProjectTree 不再默认展开所有 Volume/Arc，仅默认展开当前卷/弧；新增长篇树 SSR 回归测试。Runtime 日志既有 500 条 Store 上限，UI 只渲染末 120 条。
- 当前定向/全量验证：Go `go test ./... -count=1` 1019 passed，`go vet ./...` passed；Frontend 40 tests passed，`npm run build` passed。

## Session: 2026-09-24 — M7 final verification

- 补回 `desktop/frontend/dist/.gitkeep`，避免前端/Wails 生成目录的构建副作用进入提交。
- M7 源码审计结论、跨进程 Core book lease、Import event-before-ack、长篇 ProjectTree 默认折叠和发布文档已记录；没有新增 Core 语义或第二套锁协议。
- 最终证据：Go 1019 tests、Go vet、Frontend 40 tests、Frontend build、Windows Wails production build 均通过；GitNexus 已重建并完成非 partial/non-truncated `detect-changes`，结果为 high risk，已按动态边界保守记录。
- 发布前仍需人工/现场完成真实模型与 GUI 全流程、跨进程 CLI/GUI 同时写入、中文/空格/长路径/权限异常，以及 100/300/500/1000 章实际项目压力检查；这些未被静态测试冒充为已完成。
