# V1 范围速查

## V1 必须保留的 ainovel 能力（后续逐步 GUI 化）

- Quick Start / Co-create / Start from Outline
- 自动连续生成
- Volume → Arc → Chapter
- Architect / Writer / Editor / Arbiter
- Pause / Resume / Stop
- `/review on|off`
- `/next`
- Steer
- 人工正文编辑 + `/sync --check` + `/sync`
- Review / Rewrite / Polish
- Import
- Export TXT / EPUB
- Provider / Model / Reasoning Effort
- 角色分模型（以 Core 实际 config 为准）
- Usage / Token / Cost / Budget
- Checkpoint / Crash Recovery
- Simulation / Reference（如果当前 Core 仍支持）

## V1 UI 状态

Engine:

```text
Idle
Running
Paused
WaitingReview
WaitingSync
Error
```

Chapter:

```text
Clean
Modified
SavedUnsynced
Syncing
Synced
Generating
ReviewPending
Error
```

关键不变量：

```text
SavedUnsynced
=> WaitingSync
=> Continue Disabled
=> Next Disabled
```

## Writer Pipeline

必须读取 Core 实际实现。当前设计基线预期为：

```text
novel_context
read_chapter
plan_chapter
draft_chapter
check_consistency
commit_chapter
```

若当前源码不同，以源码为准，并同步 UI ViewModel，而不是修改 Core 去迎合设计稿。

## V1 不实现

- Proposal
- Fact Engine
- Knowledge State
- Dependency Graph
- ContextManifest
- Impact / Repair
- Advanced Evidence Diff
- Cloud Sync
- Collaboration
- Plugin Marketplace
