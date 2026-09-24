# Novel Studio V1 发布检查清单

## 基线与源码

- [ ] 发布提交来自已冻结的 M2–M7 基线，未混入 V2 功能。
- [ ] `docs/studio-m*.md`、`findings.md`、`progress.md` 与实际代码一致。
- [ ] GitNexus 已对当前提交重建；提交前运行 `detect-changes --scope all`，不得存在未解释的 `UNKNOWN`。

## Core / Studio 回归

- [ ] Quick Start、Outline 文件 prompt、Co-create、Co-create recovery。
- [ ] 连续创作、Pause/Resume、Stop、异常恢复；Pausing/Stopping 仅在 Core Done 后落终态。
- [ ] Review Center、Advance Gate、继续下一章；WaitingSync、PendingCommit、PendingRewrite、AdvanceHold 均由 Core gate 拦截。
- [ ] Running/Paused/WaitingReview Steer 路由与 Arbiter pending/error 恢复。
- [ ] Save → SavedUnsynced → Sync → Store 复核 → Synced；空白正文拒绝且原文件不变。
- [ ] Import UTF-8/GB18030 文本与 checkpoint recovery；不宣传 EPUB/PDF/DOCX Import。
- [ ] TXT/EPUB Export、Provider/API Key 脱敏、Model/Thinking、Budget、Usage。

## 并发、身份与文件系统

- [ ] GUI/CLI 同项目写入复用 Core book lease；Save/Sync/Run/Import/Next/Steer/Continue/Create/Publish 不绕过 Core 写锁。
- [ ] 项目切换覆盖 projectId、generation、sequence、requestId；迟到响应/事件不会污染当前项目。
- [ ] Quick Start、Co-create、Import 均覆盖 event-before-ack；Core 成功与视图刷新失败分离。
- [ ] 验证中文、空格、长路径、跨盘路径，以及只读/权限/临时 rename 失败。
- [ ] 检查日志、事件、错误和配置快照不含完整 API Key。

## 构建与 CI

- [ ] `go test ./... -count=1`
- [ ] `go vet ./...`
- [ ] `cd desktop/frontend && npm ci && npm test && npm run build`
- [ ] Windows runner 完成 Wails production build，并核对 `cmd/novel-studio/build/bin/Novel-Studio.exe`。
- [ ] GitHub Actions Ubuntu/Windows checks 全绿。
- [ ] 归档发布包时排除 `node_modules`、前端 `dist` 临时产物、设计源 ZIP 和完整 API Key。

## 发布包人工签收

- [ ] 从干净 checkout 构建并启动一次桌面程序。
- [ ] 选择现有项目只读打开，不因 OpenProject 创建 Host。
- [ ] 完成一次最小可恢复写作与保存/同步闭环。
- [ ] 记录构建提交、版本、产物 SHA-256、验证命令和未完成的已知限制。
