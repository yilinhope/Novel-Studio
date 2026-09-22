# Novel Studio 开发记录

## 当前范围

按交接包执行 M0 → M1 → M2 的首条只读闭环：打开已有小说项目、总览、卷/弧/章节树、正文读取。设计基准是 AI Novel Studio 的本地交接快照。暂不接入生成、编辑、Sync 等动作。

## 源码审计

- 基线提交：`3390e34e0a7756403977ad14f456a174c10b307f`。
- Go module：`github.com/voocel/ainovel-cli`，声明 Go 1.25.5。
- CLI：`cmd/ainovel-cli/main.go`；交互入口 `internal/entry/tui`，无人值守入口 `internal/entry/headless`。
- Engine 实现在 `internal/host/engine.go`，是 Host 包内的私有类型；`internal/flow` 负责路由和推进规则。
- Host：`internal/host/host.go`；Domain：`internal/domain`；配置：`internal/bootstrap`。
- Store：`internal/store`。`NewStore` 构造只读使用的对象，不调用 `Init`、Host 构造或迁移逻辑。
- 总览：`Book.Load()`、`Progress.Load()`；对应 `meta/book.json`、`meta/progress.json`。
- 树：`Outline.LoadLayeredOutline()` 和 `Outline.LoadOutline()`；对应 `layered_outline.json`、`outline.json`。分层章号按 Core 的 `FlattenOutline` 顺序连续编号。
- 章节：`Drafts.LoadChapterText()` 读取 `chapters/%02d.md`；章节规划由 `LoadChapterPlan()` 读取 `drafts/%02d.plan.json`。
- `VolumeOutline`、`ArcOutline`、`OutlineEntry` 在 `internal/domain/story.go`。
- 长篇 `Progress.TotalChapters` 是内部估算，不得显示为固定目标；界面显示实际已规划章数。
- TUI command handler 含参数校验和终端视图操作；Review、Next、Steer、Sync、Import、Export 的业务入口已在 Host。后续 M3 应适配 Host 事件与生命周期，而非移植终端组件。
- 主要依赖：Bubble Tea、Lip Gloss、agentcore、litellm。桌面层新增 Wails，前端使用 React、TypeScript、Zustand。

## 验证记录

系统 PATH 没有 Go。已从 Go 官方发行站下载并校验 SHA256，将 Go 1.27.1 放在被忽略的 `.cache/toolchain/go`，未修改系统 PATH。当前 Go 测试、`go vet`、前端测试、前端构建和 Wails 生产构建均已通过；`npm audit` 为零漏洞。

浏览器验收已覆盖：欢迎页、项目总览、卷/弧/章节树、已保存正文读取、空正文章节状态，以及无 Wails bridge 时的明确错误提示。原生桌面窗口的人工点击验收受当前 Computer Use 接口不可用影响，已用生产 Wails 构建和浏览器 bridge mock 完成可重复验证。

## 实施布局

新增 `internal/studio/app` 与 `viewmodel`，Wails 入口位于 `cmd/novel-studio`；资源嵌入包及 React 工程位于 `desktop/frontend`。Wails 配置随桌面入口放置，以便保持单一 Go module。

## 后续 M3

接入 Host 生命周期和事件，统一派生 Engine 状态，补充 Continue/Pause/Resume/Stop 与运行日志。持久化 Phase/Flow 不能证明引擎当前正在运行，M2 界面仅声明只读浏览。
