# `novel_studio_6` 使用说明

该目录代表 **Chapter Editor — Waiting Sync / Unsynced**。

- `code.html` 中的 Waiting Sync / 立即同步 / Continue Disabled 逻辑可作为参考。
- 当前 `screen.png` 仍可能保留旧状态表现，**不要按该截图实现业务状态**。
- 实现时以 `docs/Novel_Studio_V1_Design_Baseline_Annotated.md` 中的 WaitingSync 规则为准。

正确规则：

```text
SavedUnsynced
→ EngineWaitingSync
→ Continue Disabled
→ Next Disabled
→ Sync
→ Synced
```
