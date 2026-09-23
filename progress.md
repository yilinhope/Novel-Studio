# Progress Log

## Session: 2026-09-23 — M4-A Review 与 PR

- **Status:** complete
- 用户要求按不依赖 gstack checklist 的替代流程完成人工 review 后创建 PR；缺失 checklist 不再阻断本轮。
- 手工审查 M4-A 全量源码后发现审计方案中的 Resume 门禁尚未接入 UI；已补充 RuntimeCenter 禁用态、原因提示和 Engine Store action 级防护，未知/检查中/错误/未同步状态均不会调用 ResumeWriting。
- GitNexus 编辑前影响：RuntimeCenter LOW（1 个调用者、0 条图流程）；`resumeWriting` 查询 UNKNOWN（多候选/调用关系解析不足），已用 rg 核实按钮与 action 调用边界。
- 补齐后验证通过：`go test ./...`、`go vet ./...`、Vitest 13/13、`npm run build`、Windows/amd64 Wails production build、`git diff --check`。
- Vite/Wails 构建移除了已跟踪 `desktop/frontend/dist/.gitkeep`；已恢复基线 1 字节占位文件。
- 暂存后的 GitNexus 检查完成：17 个文件、108 个符号、0 个受影响流程、LOW；图索引含 node_modules/Vite 产物并有第三方流程截断，流程结果仍按下界解读。
- `git diff --cached --check` 首次发现审计 Markdown 元信息行的两个尾随空格；已改为列表并复查通过。
- 当前人工 review 未发现未解决问题。提交 `1130fa8`（M4-A：实现只读章节修订状态检查）已推送；Draft PR #3 已创建并关联到当前 Codex 任务：https://github.com/yilinhope/Novel-Studio/pull/3。
- GitHub Connector PR 创建鉴权失败，确认分支无既有 PR 后按回退流程由已登录 `gh` 创建；PR 创建时 checks 为 pending/0 项，待 GitHub Actions 刷新。

## Session: 2026-09-23 — M4 章节编辑与同步 Core 审计

- **Status:** awaiting_confirmation
- 已读取 M4 规格要求；本轮限制为先核查当前 Core 修订链并输出 M4-A 实施方案。
- 确认 PR #2 已合并，merge commit `8e815ad2beae8a30063f8f80c991ee64923e1713`；从更新后的 `origin/main` 创建 `codex/m4-chapter-editing-sync`。
- 刷新 GitNexus 到 9,469 nodes、39,216 edges、330 clusters、669 flows。流程提取有入口及分支截断；审计报告明确按下界处理。
- 分析 `revision.Scan`（CRITICAL）、`Host.CheckChapterRevisions`（HIGH）、`Host.Resume`（HIGH）、`Host.acquireExclusive`（CRITICAL）、`Host.SyncChapterRevisions`（LOW）、`revision.Service.Sync`（下界 LOW）和 `Host.AdvanceOneChapter`（UNKNOWN）；对 UNKNOWN/接收者解析遗漏通过仓库文本搜索复核。
- 逐项核对 18 个问题：扫描只读语义、哈希基线、批量 Sync、Projector 派生状态、三阶段崩溃恢复、继续/下一章 gate、Studio OpenProject 只读边界、Host lease 与 Store 实例锁、暂停 Done 时序。
- 新增 `docs/studio-m4-revision-audit.md`，含代码级调用链、影响风险、并发缺口和 M4-A 文件/API/测试方案。
- 本轮未修改产品源码，未运行测试；M4-A 等待用户确认审计结论。

## Session: 2026-09-23 — M4-A Revision Check

- **Status:** complete
- 用户已明确批准 M4-A，并确认范围仅包含 Revision ViewModel、只读 RevisionService、两个只读 API 和前端 revisionStore。
- 已将 M4-C 按需创建 Host 与 M4-B 统一项目写入互斥列为强制约束；M4-A 不引入 Save 或 Sync。
- 当前分支为 `codex/m4-chapter-editing-sync`；上一轮审计文档和计划文件为本工作树既有改动，予以保留。
- GitNexus 刷新及编辑前影响分析完成；`currentProject` HIGH（只读调用）、RuntimeState/Bridge 动态绑定 UNKNOWN（已源码检索核实），前端 Store `open` / `selectProject` 为 UNKNOWN 下界结果（调用站点已检索）。
- 已增加 RevisionStatus 状态/哈希/pending-stage ViewModel、只读 RevisionService、Bridge Get/Check API、前端 revisionStore，并接入打开项目后的 Store 检查；新增 Go/前端测试。
- 首次 `go test ./internal/studio/...` 暴露空章节 slice 缓存复制为 nil 的一致性问题；已修复为保留空数组语义，后续 Go 测试全部通过。
- `npm ci` 成功，安装 97 个依赖，审计报告 0 vulnerabilities。
- 已实现 ViewModel `RevisionStatus`（unknown/synced/saved_unsynced/recovery_pending/error）、只读 RevisionService、hash/pending stage 返回、读取最近检查状态缓存；扫描错误保留 error 和保守 `hasUnsynced`，不误报 synced。
- Bridge 暴露 `GetRevisionStatus` / `CheckChapterRevisions`；检查只读加载 pending 或调用 `revision.Scan`，Running/Pausing/Stopping 时拒绝。API 不调用 `Host.New`；bridge 测试确认 GetChapter + Revision Check 后 Host 仍未创建。
- 检查期间以 App 项目控制锁串行化 OpenProject/Resume，避免检查与并发 Resume 或项目切换交错；Store 文件检查本身不写数据。
- 前端新增独立 `revisionStore`，项目选择时先读缓存再执行真实 Store Check，使用项目 ID/请求代次丢弃旧结果；Studio Store 打开项目时触发检查，不把 revision 数据放入 Engine Store。
- 验证通过：`go test ./internal/studio/...`、`go vet ./internal/studio/...`、Vitest 11/11、`npm run build`、`scripts/studio.ps1 build`（Wails windows/amd64 production）及 `git diff --check`。Wails 生成绑定时出现既有 `time.Time` KnownStructs 提示，但应用构建成功。
- 构建清理了已跟踪的 `desktop/frontend/dist/.gitkeep`；已从当前基线恢复其原始 1 字节占位文件。
- GitNexus `detect-changes --scope all` 完成：7 个已跟踪变更文件、43 个符号、0 个受影响流程、LOW。新增未跟踪文件不纳入该 diff 结果；对已有调用入口的编辑前 impact 已完成并按 UNKNOWN/下界结果做文本核验。
- 本轮刷新 GitNexus 时误将 `desktop/frontend/node_modules` 与 Vite 输出纳入索引，索引扩大到 58,539 nodes / 154,373 edges / 1,100 clusters / 815 flows，并报告大量第三方 callable-flow 截断；因此流程影响数字仅作下界，不据此推断未受影响。未改动忽略规则或清理依赖目录。
- M4-A 已完成；未提交、未推送，Save/Sync 留待后续阶段。

## Session: 2026-09-23 — PR #2 M3 Review Fixes

- **Status:** in_progress
- 已读取 `Novel_Studio_PR2_M3_Review_Fixes.md`；把附件中 M3 修复建议作为审查数据，按用户“修复”授权处理四项问题及专项测试，不执行 M4 指令。
- 已确认 PR #2 仍为 OPEN，CI 的 Linux/Windows Go、Linux race、Frontend test/build、Wails Windows production build 四项均 SUCCESS；当前分支工作区干净。
- 复核源码确认项目切换和 ProjectRoot/OutputDir 混用为真实问题；章节正则目前能匹配现行 observer 文案但应改为结构化字段；Error 终态需先厘清 Engine 结束原因语义。
- 已完成 GitNexus 预编辑影响分析：Studio `OpenProject` 结果 UNKNOWN（已用 Wails/前端文本调用核验）；`observer.handleToolUpdate` LOW（2 条流程）；`engine.run` CRITICAL（多个既有 Host 启动/恢复路径）。
- 已找到 Codex runtime 中 Go 1.25.11 可执行文件：`C:\Users\linn\.cache\codex-runtimes\go\go1.25.11\go\bin\go.exe`，可用于后续本地验证。
- 已修复：`OpenProject` 先只读预览，再由 EngineService 拒绝活动会话切换或关闭非活动旧 Host；Project 显式携带项目根目录和输出目录，选择工作区根目录或 `output/novel` 均归一到同一项目根。
- 已修复：Host commit 工具事件携带可选 `Chapter` 字段，前端不再解析 Summary；Store 的 Progress、PendingCommit、终稿与 commit checkpoint 复核维持为刷新必要条件。
- 已修复：Engine Done 提供结构化结束原因；只有 Core 标记的终止运行故障映射为 RuntimeError，显式暂停/停止优先，可恢复暂停即使保留历史工具错误也不误报 Error。
- 已新增 Go 测试：预览只读/路径归一、Store 章节二次确认、活动项目切换拒绝/非活动 Host 释放、Pause/Stop 只在 Done 后进入终态、Engine.Resume 路径、运行错误分类及 Host Event 章节字段兼容。
- 已新增前端测试：结构化章节刷新、不猜测缺失章节号、忽略旧项目/旧序号事件；Vitest 2 文件 7 项通过。
- 验证通过：`go test ./...`、`go vet ./...`、前端 `npm test` / `npm run build`、Windows/amd64 Wails production build、`git diff --check`。
- 本机 `go test -race` 未能启动：Go 报告需要启用 CGO；环境内无 gcc/clang/zig。推送后 GitHub race job 已通过。
- GitNexus 刷新到 9,469 nodes、39,216 edges、330 clusters、669 flows；`detect-changes --scope all` 报 17 个文件、97 个符号、57 条流程、CRITICAL。影响集中在 Engine/Host Event/Studio Bridge 共享路径；已逐项复核为新增结束原因和可选结构化字段，不改 TUI 路由及 Core Engine 调度语义。
- 已提交 `08ff7f0`（修复：补齐 M3 项目切换与 Engine 终态边界）并推送到 PR #2；PR 描述已同步更新，所有 GitHub Checks（Ubuntu/Windows Go、Frontend、Windows Wails）均通过。
- **本阶段状态：**修复、验证、提交、推送与 PR 更新全部完成。

## Session: 2026-09-23 — M3 Engine Bridge 审计

- **Status:** complete（M3-A / M3-B / M3-C 已分段提交）
- 已读取 `Novel_Studio_V1_M3_Engine_Bridge_Codex.md`，本轮按其第一步执行源码审计和 M3-A 实施方案。
- 从 M2 创建 `codex/m3-engine-bridge-audit`，开始时工作树干净。
- 已刷新 GitNexus 索引，并定位 `Host.New`、`Host.Resume/Continue/Abort`、TUI `resumeBook`、`Host.Events` 和 `Host.Snapshot`。
- 已核对生命周期、review gate、usage、日志、commit 信号与 M2 边界；源码级结论和 M3-A 实施方案写入 `docs/studio-m3-engine-audit.md`。
- 审计已经用户确认；目前按 M3-A 实施，Wails Event Bus 与控制 UI 分别留到 M3-B / M3-C。
- 用户已确认审计通过并授权进入 M3-A，要求 Studio 继续写作调用 `Host.Resume()`、增加 Pausing/Stopping、只在 Core Done 后确认终态、OpenProject 保持只读，并按 M3-A/B/C 分段提交。
- 已新增 `internal/studio/app/engine_service.go`、`internal/studio/viewmodel/runtime.go`，并扩展 `bootstrap.LoadConfigFromDir`；M3-A 仍未接入 Wails Event Bus 或前端。
- GitNexus 对 `bootstrap.LoadConfig` 上游影响为 LOW（ainovel-cli main 单一调用）；对 Host Event 字段为 CRITICAL（35 个直接依赖），所以本阶段没有修改 Host Event。
- Go 工具链未在 PATH：先后核对常见安装位置后，在 Codex runtime cache 找到 Go 1.25.11。`gofmt` 和 `go build ./internal/studio/... ./internal/bootstrap` 最终通过；没有运行测试。
- 已通过 `go build ./...` 与 `git diff --check`；GitNexus `detect-changes --scope all` 报告 LOW、0 个受影响流程。下一步提交 M3-A 阶段。
- M3-A 已按独立提交完成：`05ca87f feat: 增加 Studio Engine Session 与 Runtime ViewModel`；工作树干净。现在进入 M3-B，沿用 handoff 里 Running/Paused/Error Runtime Center 设计。
- M3-B 的 `host.Event` 上游影响经新索引确认 CRITICAL：79 个受影响符号、35 个直接依赖、31 条流程、6 个模块。字段只用于新增 Studio 投影；TUI/Engine 逻辑不改。
- M3-B 已接入 Wails `studio:engine-event` 转发、项目代次/序号过滤的 Zustand Engine Store、Runtime Center（Runtime / Agent / Writer Tool / 用量 / 最近日志）。章节完成后的 Store 复核与控制按钮仍留在 M3-C。
- M3-B 验证：`go build ./...` 通过；`npm ci` 后 `npm run build` 的 TypeScript 检查与 Vite production build 通过；未运行测试套件。构建产物及本轮 `node_modules` 已清理，`desktop/frontend/dist/.gitkeep` 已恢复。
- GitNexus 重新索引为 9,301 节点、38,551 关系、316 clusters、664 flows；其流程枚举仍提示截断。精确 `Event` 影响报告仍为 CRITICAL（79 符号/35 直接/31 flows），但 TUI 不读取新增 `Tool` 字段，Engine 路由未改；MCP 查询确认 Studio monitor 事件消费路径，并完成 M3-B staged 变更检查。
- M3-B 已独立提交：`8cd53d8 feat: 接入 Studio Runtime 事件中心`。GitNexus staged 检查 16 文件/211 符号、23 条流程、CRITICAL；影响集中在 Host.Event 共享类型，已按可选 Tool 字段边界检查 TUI/Engine 无消费或路由变化。
- 进入 M3-C：开始前重新查询 Host 控制与 Studio 项目/章节读取符号影响，实施控制操作及 Store 二次确认刷新。
- M3-C 已实现 `PauseWriting` / `ResumeWriting` / `StopWriting`。Studio 继续创作仍只调用 `Host.Resume()`；Pause/Stop 先呈现 Pausing/Stopping，只有 monitor 收到 Done 后发布 Paused/Stopped；Stop 完成后关闭 Host 并释放目录租约。控制操作由 control mutex 串行化，避免 Resume/Abort/Done 竞态。
- M3-C 章节刷新只响应成功的 `commit_chapter` 工具事件；Store 新建磁盘快照并确认 Progress 包含目标章、PendingCommit 已清除、终稿非空、对应 commit checkpoint 在本次工具开始后生成，才返回刷新项目与当前章节。
- M3-C 验证：`go build ./...`、`go vet ./internal/studio/... ./internal/host`、前端 `tsc --noEmit` + Vite production build、Wails v2.15 Windows/amd64 production build 均通过；未运行测试套件。Wails 绑定生成输出有 `time.Time` 未找到提示，但生产构建成功。
- M3-C GitNexus：`Host.Resume` HIGH（10 符号/3 流程，既有 TUI 调用链；本阶段只新增 Studio 调用、不改 Host/TUI 行为）；`Host.Abort` LOW（3 符号/1 流程）。当前 staged 前全量变更 11 文件/46 符号/22 流程，CRITICAL 由 Engine/Host 事件路径带入，已复核为明确调用边界和只读 Store 检查；GitNexus 索引 9,385 节点、38,857 关系、325 clusters、667 flows，流程枚举有截断警告。
- M3-C 已独立提交；三阶段按序为 M3-A → M3-B → M3-C，各阶段保持独立提交。

## Session: 2026-09-22

### Phase 1: 项目上下文恢复

- **Status:** complete
- **Started:** 2026-09-22
- Actions taken:
  - 确认项目根目录和当前分支。
  - 检查计划文件不存在，并读取 planning-with-files 技能规则与模板。
  - 根据既有交接与开发记录恢复 M2、review、验证和 PR 上下文。
- Files created/modified:
  - `task_plan.md`
  - `findings.md`
  - `progress.md`

### Phase 2: M2 review、验证与 PR 状态固化

- **Status:** complete
- Actions taken:
  - 记录 M2 代码 review 结论和唯一实质问题：干净克隆缺少 `dist` 目录。
  - 记录 `dist/.gitkeep` 修复、GitNexus 索引恢复和工作树清洁状态。
  - 记录 PR #1、分支和两个已交付提交。
- Files created/modified:
  - 计划文件三件套
  - M2 代码和交接包已在 `88cae4f`、`97f698d` 中交付

### Phase 3: M3 路线定义

- **Status:** complete
- Actions taken:
  - 将 Engine、Sync、Review、Steer、Import、Export 明确列为后续 M3 范围。
  - 将写入确认、失败恢复、并发保护、脏状态提示列为 M3 设计前置问题。
  - 保留真实 Core Store、GitNexus 影响分析和原生桌面验证边界。
- Files created/modified:
  - `task_plan.md`
  - `findings.md`

### Phase 4: 计划文件交付

- **Status:** complete
- Actions taken:
  - 在项目根目录创建 `task_plan.md`、`findings.md`、`progress.md`。
  - 将本次 PowerShell 引号错误和 catch-up 路径差异写入错误记录。
- Files created/modified:
  - `task_plan.md`
  - `findings.md`
  - `progress.md`

### Phase 5: PR 卫生与 CI 补强（进行中）

- **Status:** in_progress
- **Started:** 2026-09-22
- Actions taken:
  - 核对 PR #1 当前 head、远端分支和 GitHub checks 状态。
  - 确认 Go CI 已存在，但前端测试/构建和 Wails Windows production build 尚未纳入门禁。
  - 确认嵌套 Stitch ZIP 只在 `97f698d` 引入，且尚未合并到 `main`，具备清理 PR 历史的窗口。
  - 将章节树的长篇性能问题拆为后续测量与优化阶段。
- Files created/modified:
  - `task_plan.md`
  - `findings.md`
  - `progress.md`

### Phase 5 Errors

- 首次读取 planning 文件时 PowerShell 外层引号剥离变量；已改用独立单引号命令。
- 旧版 catch-up 脚本路径不存在；已记录环境差异，未阻断当前工作。

### Phase 7 Completion Notes

- 已在临时 worktree 从 `main` 重建 PR 分支，重新应用 M2、精选 handoff 和计划文件提交。
- 已删除 `Novel_Studio_Codex_Handoff/design/source/stitch_ai_novel_studio_latest.zip`，并同步更新 README、MANIFEST 的引用与校验值。
- 已在 `.github/workflows/ci.yml` 增加前端 `npm ci`、`npm test`、`npm run build` job，以及 Windows Wails production build job。
- 原始 PR 分支留有本地备份引用，待新分支验证并强制更新远端后再清理。
- 已完成 `--force-with-lease` 更新，远端 PR head 为 `ad50480`；本地旧备份和临时 worktree 已清理。
- GitHub Actions 权限 API 返回 enabled，但 workflow 列表为 `total_count: 0`，PR 仍显示 `Checks 0`；该项保留为外部平台阻塞。

## Test Results

| Test | Input | Expected | Actual | Status |
|------|-------|----------|--------|--------|
| Go 测试 | `scripts/studio.ps1 test` | 全部通过 | Go suite 全部通过 | ✓ |
| Vitest | `scripts/studio.ps1 test` | 前端测试通过 | 4/4 通过 | ✓ |
| Vite 构建 | `scripts/studio.ps1 test` | 生产前端构建通过 | 1587 modules transformed，成功 | ✓ |
| Go vet | `go vet ./internal/studio/...` | 无诊断 | 通过 | ✓ |
| Wails 构建 | `scripts/studio.ps1 build` | Windows 生产构建通过 | Wails v2.15.0 成功 | ✓ |
| 干净归档测试 | staged archive + `go test ./...` | 干净克隆可测试 | 通过 | ✓ |
| 浏览器 QA | Playwright mock bridge | 核心状态可用 | 欢迎、概览、树、正文、空状态、错误态通过 | ✓ |
| GitNexus | `gitnexus status` | 索引与当前提交一致 | up-to-date | ✓ |
| GitHub checks | `gh pr checks 1` | 获取 CI 状态 | 当前无 checks 报告 | — |
| 清理后 ZIP 路径 | `git rev-list --objects HEAD` | 不包含嵌套 ZIP | 新分支 HEAD 不包含 | ✓ |
| Workflow lint | `actionlint .github/workflows/ci.yml` | YAML/Action 语法通过 | 待工具可用性检查 | — |
| GitHub workflow registration | `gh api repos/yilinhope/Novel-Studio/actions/workflows` | 至少发现 CI workflow | `total_count: 0`，平台层待处理 | — |

## Error Log

| Timestamp | Error | Attempt | Resolution |
|-----------|-------|---------|------------|
| 2026-09-22 | PowerShell 外层引号剥离变量，导致批量读取命令 ParserError | 1 | 改用独立单引号 `-Command` 命令 |
| 2026-09-22 | 技能指定的旧版 catch-up 脚本路径不存在 | 1 | 记录环境差异，使用当前仓库状态继续初始化 |

## 5-Question Reboot Check

| Question | Answer |
|----------|--------|
| Where am I? | M2 已完成，计划文件交付完成；M3 pending |
| Where am I going? | 先确定 M3 首个写入闭环，再做影响分析、实现和回归验证 |
| What's the goal? | 建立可跨会话恢复的 Novel Studio 开发计划并维护 M2/M3 交付边界 |
| What have I learned? | 见 `findings.md`：真实 Core Store、GitNexus、浏览器 QA 和原生验证边界 |
| What have I done? | 创建并填写 `task_plan.md`、`findings.md`、`progress.md` |

---

*后续每完成一个阶段或遇到错误，都要同步更新本文件。*
