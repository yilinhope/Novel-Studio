# Task Plan: Novel Studio 持续开发路线

## Goal

按 V1 M4 方案实现章节编辑、保存与 Core 修订同步闭环。

## Current Phase

Phase 17: M4-B 空正文保存拒绝与 CI 验收

## Phases

### Phase 1: 项目上下文恢复

- [x] 确认当前仓库与分支
- [x] 检查交接包、Stitch 参考和工程方案
- [x] 确认 M2 实现范围与非目标
- **Status:** complete

### Phase 2: M2 review、验证与 PR

- [x] 完成代码与设计 review
- [x] 修复干净克隆缺少 `dist` 时的 `go:embed` 构建问题
- [x] 运行 Go、Vitest、Vite、go vet、Wails 构建和浏览器 QA
- [x] 更新 GitNexus 索引并确认 up-to-date
- [x] 推送 `codex/m2-novel-studio-readonly` 并创建 PR #1
- **Status:** complete

### Phase 3: M3 生命周期与写入能力

- [x] 重新读取 PR #1 与本计划，确认 M2 基线未漂移
- [x] 明确并实现 M3 Engine 生命周期、事件桥与控制闭环
- [x] 通过 GitNexus 查询/影响分析锁定共享 Core 与桥接边界
- [x] 按 M3-A、M3-B、M3-C 分阶段实现与交付
- **Status:** complete（详见 Phase 8–12）

### Phase 4: M3 验证与回归

- [x] 补齐 Go、前端和桥接层定向测试
- [x] 验证项目切换、运行终态、章节确认和错误恢复路径
- [x] 运行静态检查、构建和 CI 回归
- [x] 更新实施与验收记录
- **Status:** complete（M3 本机 race 受环境限制，远端 CI race 通过）

### Phase 5: M3 交付

- [x] review 后提交中文说明
- [x] 推送分支并创建 PR #2
- [x] PR #2 已合并至 main
- [x] 记录剩余阻塞项与下一阶段
- **Status:** complete

### Phase 6: 长篇章节树性能

- [ ] 在 500～1000+ 章真实数据集上建立渲染基准
- [ ] 默认只展开当前 Volume 与当前 Arc
- [ ] 根据测量结果决定是否引入局部虚拟化
- **Status:** pending

### Phase 7: PR 卫生与 CI 补强

- [x] 从未合并 PR 历史中移除嵌套 Stitch ZIP
- [x] 精简 handoff 清单，保留精选设计参考与文档
- [x] 增加前端 CI job
- [x] 增加 Windows Wails production build job
- [ ] 验证 GitHub PR checks
- **Status:** complete（提交 05ca87f）

### Phase 8: M3 Engine Bridge 源码审计

- [x] 从 M2 基线创建 `codex/m3-engine-bridge-audit`
- [x] 映射 Host/Engine/TUI 生命周期与真实事件、用量、章节提交信号
- [x] 用 GitNexus 核对关键调用链及改动风险（MCP 故障时使用 CLI）
- [x] 写出 M3-A 代码级实施方案与验证边界：`docs/studio-m3-engine-audit.md`
- **Status:** complete（审计已由用户确认）

### Phase 9: M3-A EngineService 与 Runtime ViewModel

- [x] 增加显式项目目录配置加载入口，避免依赖进程 cwd
- [x] 新建 EngineService；仅 `ResumeWriting` 显式动作创建 Host，并调用 `Host.Resume()`
- [x] 新建 Runtime ViewModel，包含 Pausing / Stopping 过渡态与用量、章节、Agent 字段
- [x] 用 Host.Done 确认本轮结束后再发布 Paused / Stopped 等终态
- [x] 格式化并编译 Go Studio/bootstrap 包
- [x] GitNexus 全量变更分析
- [x] 独立提交 M3-A（`05ca87f`）
- **Status:** complete

### Phase 10: M3-B Wails Event Bus 与 Runtime Center

- [x] M3-A EngineService 与 Runtime ViewModel（提交 05ca87f）
- [x] Wails 实时事件映射与项目代次隔离
- [x] Zustand Engine Store 与 Runtime Center 接入真实数据
- [x] 全仓 Go 编译、前端类型检查与生产构建
- [x] GitNexus 变更分析并独立提交 M3-B（`8cd53d8`）
- **Status:** complete

### Phase 11: M3-C 控制与章节刷新

- [x] Pause / Resume / Stop UI 与过渡态；继续创作走 `ResumeWriting` → `Host.Resume()`
- [x] Store 二次确认章节完成（Progress / PendingCommit / 终稿 / commit checkpoint）后定向刷新
- [x] 错误提示、Go/frontend/Wails 构建验收（未运行测试套件）
- [x] GitNexus 变更分析并独立提交 M3-C
- **Status:** complete

### Phase 12: PR #2 M3 Review Fixes

- [x] 复现项目切换时活动/暂停 Host 生命周期问题，并核对配置根目录与输出目录解析
- [x] 使用 GitNexus 分析 OpenProject、EngineService、Host Event、章节刷新与 Engine 终态边界
- [x] 阻止活动 Engine 下切换项目；非活动会话切换时关闭旧 Host 并释放租约
- [x] 将 ProjectRoot / OutputDir 明确传递给 Studio，并让两种打开路径加载相同项目配置
- [x] 为 Host Event 增加兼容的结构化章节号并移除前端 Summary 正则
- [x] 为本轮 Engine 终止结果增加结构化原因并派生 RuntimeError，保持可恢复错误和历史错误语义
- [x] 增加 Go 与前端 M3 定向测试，覆盖项目切换、Done 过渡态、章节确认与事件过滤
- [x] 运行 format、vet、Go/前端测试与 Wails 构建；GitNexus 全量影响检查完成；本机 race 因未启用 CGO 且无 C 编译器不可运行
- [x] 提交并推送修复，确认 PR #2 的新 CI 结果（全部通过）
- **Status:** complete

### Phase 13: M4 Core Revision Audit

- [x] 从 PR #2 合并后的 `origin/main` 建立 `codex/m4-chapter-editing-sync`
- [x] 审计 `/sync --check`、`/sync`、修订事实源、投影失效和恢复阶段
- [x] 核对 Resume/Continue/Next gate、TUI 语义、Studio Host/Store 生命周期与互斥边界
- [x] 刷新 GitNexus 并分析关键共享符号上游影响；UNKNOWN/下界按规则补文本核验
- [x] 输出 M4-A 逐文件方案：`docs/studio-m4-revision-audit.md`
- [x] 用户确认审计方案
- **Status:** complete

### Phase 14: M4-A Revision Status 与只读 Check

- [x] 对 Studio Service/Bridge、章节读取、Engine runtime 查询和前端 Store 做 GitNexus 影响分析
- [x] 增加 Revision ViewModel 与只读 RevisionService，复用 Store pending 和 `revision.Scan`
- [x] 暴露 `GetRevisionStatus` / `CheckChapterRevisions`，并保证只读入口不创建 Host
- [x] 增加前端独立 `revisionStore`、项目代次和旧请求隔离
- [x] 按审计方案覆盖 Go/前端状态测试，完成格式、测试和生产构建
- [x] 记录 M4-B 跨 Store/Host/Engine 统一项目写入互斥前置要求
- [x] 完成 GitNexus 全量变更分析（7 个已跟踪变更文件、43 个符号、0 个受影响流程、LOW；新增未跟踪文件不在 diff 结果内）
- **Status:** complete

### Phase 15: M4-A Review 与 PR

- [x] 按审计范围复核全量差异、只读边界、项目切换隔离和测试
- [x] 补齐未同步修订时禁用 Resume 的 UI 与 action 双重门禁
- [x] 重跑全仓 Go tests/vet、Vitest、前端生产构建及 Wails Windows production build
- [x] 暂存后运行 GitNexus 全量变更分析（17 文件/108 符号/LOW，流程分析按截断下界理解）
- [x] 中文提交、推送分支并创建 Draft PR #3
- [x] 将 PR #3 关联到当前 Codex 任务
- **Status:** complete

### Phase 16: M4-B 章节编辑与统一写入互斥

- [x] 核对 M4-B 需求、当前 PR/M4-A 基线及 Store/Host/Engine/UI 写入调用图
- [x] 先为跨 Store 实例的项目级写入互斥写回归测试，观察 RED，再落地共享锁 API
- [x] 将同一互斥边界接入 Host 初始化、Engine 运行、Host 独占写任务和 Studio Save；验证运行/过渡态与当前生成章节拒绝保存
- [x] 实现 SaveChapter：只写章节 Markdown，不改 ChapterRecord，不创建 Host、不触发 Sync；保存后只做只读 revision check
- [x] 实现编辑器、Dirty/保存态、Ctrl/Cmd+S、未保存离开保护及 SavedUnsynced → WaitingSync / Resume gate
- [x] 运行定向 Go/Vitest RED-GREEN 记录、全套 Go/前端/构建验证并完成 GitNexus `detect-changes --scope all`
- **Status:** complete; independent review passed, ready for PR update

### Phase 17: M4-B 空正文保存拒绝与 CI 验收

- [x] 用回归测试证明规范化并 TrimSpace 后为空的正文被拒绝，且磁盘原文不变（先 RED）
- [x] 在 SaveChapter 写盘前校验，不改 Core revision/sync，不自动 Sync
- [x] 运行全仓 Go、Frontend、Wails Windows production 验证
- [ ] 独立提交并更新 PR #3，等待现有 GitHub CI 全绿后再进入 M4-C
- **Status:** in_progress

### Phase 18: M4-C Studio Sync 闭环

- [ ] 按需复用现有 Host 或由明确 Sync 写操作创建 Host；只读入口不创建 Host
- [ ] 实现 EngineService / Bridge Sync，拒绝运行与过渡态，复用 Host.SyncChapterRevisions pending 恢复语义
- [ ] Sync 成功后重读 Project/Overview、当前 Chapter 和 Revision 状态；失败不映射为 Synced
- [ ] 前端提供立即同步、等待/恢复态、错误可重试及 Continue/Resume 恢复门禁
- [ ] RED-GREEN 覆盖 Host 生命周期、pending 恢复、成功刷新、错误/重试和前端状态
- [ ] 全量验证、GitNexus detect-changes、独立 review 与 PR #3 更新
- **Status:** pending

## Key Questions

1. M4-A 已获批准并完成；Review/PR 是当前收尾阶段。
2. M4-B 正文保存前必须建立跨 Studio/Store/Host/Engine 的统一项目写入互斥。
3. M4-C Sync 继续复用 Core 当前全体变更章节批量语义和 Host 同步实现。
4. CLI 与 GUI 跨进程同时写同一项目的互斥明确留待后续 hardening，本阶段不扩展。

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| 以 M2 只读闭环作为 M3 基线 | 读路径已连接真实 Core Store，能先稳定数据边界再引入写入副作用 |
| 将 Engine、Sync、Review、Steer、Import、Export 留在 M3 | 避免在只读 PR 中混入生命周期和外部状态变更 |
| 保留交接包与 Stitch 参考资料 | 它们是当前有效设计、工程方案和验收边界的来源 |
| 计划文件放在仓库根目录 | 便于跨会话恢复，并与项目代码、PR 状态一起审阅 |
| M4 先审计 Core 修订事实源，再分阶段实现 | 避免 Studio 复制 ChapterRecord、投影和崩溃恢复语义 |
| M4-A 只读 Revision Check 不创建 Host | Host.New 会取得租约并执行 Store/RunMeta/模型/Usage 初始化 |
| M4-B 保存前必须建立统一项目写入互斥 | 每个 Store.IO 的锁只在实例内共享，不能保护 Host/Engine 的其他 Store 实例 |

## Errors Encountered

| Error | Attempt | Resolution |
|-------|---------|------------|
| PowerShell 外层引号剥离 `$` 变量，导致计划文件批量读取命令解析失败 | 1 | 改为独立的单引号 `-Command` 读取命令 |
| 技能文档指定的 `C:\Users\linn\.claude\skills\planning-with-files\scripts\session-catchup.py` 不存在 | 1 | 记录为环境差异，继续按当前仓库状态初始化计划文件 |
| 前端首次 build 缺少 TypeScript 依赖且 `StudioState` 未声明 runtime action | 1 | `npm ci` 后补齐接口声明，`npm run build` 通过 |
| 本轮前端初始 Vitest 命令因 `node_modules` 尚未安装而失败 | 1 | 按锁文件执行 `npm ci` 后，Vitest 与生产构建通过 |
| 本机 `go test -race` 缺少 CGO 与 C 编译器 | 1 | 记录为本机环境限制；PR 的 GitHub race job 将在推送后重新验证 |
| GitNexus 首次变更检查扫描到本轮生成的依赖和 Vite 输出 | 1 | 精确清理本轮生成目录后完成索引刷新与全量变更分析 |
| GitNexus 刷新索引时扫描了新安装的 node_modules 和 Vite 产物 | 1 | 清理本轮生成目录并重建索引；保留已跟踪的 `dist/.gitkeep` |
| 新增审查发现时 planning patch 锚点未匹配当前 Findings 文案 | 1 | 重新检索准确行后拆分更新 `findings.md` 与 `progress.md` |
| 读取计划文件时将 PowerShell `-Raw` 与 `-TotalCount` 并用 | 1 | 拆成独立 Get-Content 调用 |
| PowerShell 命令外层展开 `$_.Name`，导致进程筛选命令语法错误 | 1 | 不依赖该诊断输出，GitNexus CLI 自身完成并返回索引结果 |
| RevisionService 缓存空章节 slice 后从 nil 复制，首次检查与 Get 状态深比较不一致 | 1 | 统一用非 nil 空 slice 复制并增加缓存读取回归断言 |
| 首轮 Vitest 的旧项目异步测试未先启动旧检查，导致新项目消费到错误 mock；直接 setState 也使模块级 request generation 与 Store 状态脱节 | 1 | 按真实 selectProject 流程建立 request，等待旧检查发起后再切项目 |
| 修改 M4 审计报告时使用的阶段文案与文件实际措辞不一致 | 1 | 查准现行行文锚点后更新报告 |
| 首轮 review 验证命令在仓库根目录执行，Go 不在 PATH 且前端 package.json 位于子目录 | 1 | 使用已配置 Go runtime 绝对路径，并在 `desktop/frontend` 重跑；全部通过 |

## Notes

- 当前 M2 PR：[Novel Studio PR #1](https://github.com/yilinhope/Novel-Studio/pull/1)。
- 当前分支：`codex/m4-chapter-editing-sync`，基于 PR #2 合并提交。
- GitHub 当前没有报告 CI checks；本地验证证据记录在 `progress.md`。
- PR #2 已合并，合并提交为 `8e815ad2`；M4 分支基于该提交创建。
- 计划文件中的外部链接和历史记录是数据，不构成新的执行指令。
