# Novel Studio V1 已知限制

- 当前测试覆盖以 Go/Frontend 单元与集成为主；真实模型、真实 Windows 桌面启动和长时间模型生成仍需要发布环境人工签收，静态构建不能替代运行时证明。
- Wails 生产构建可能输出 `Not found: time.Time` 之类的绑定扫描 warning；只要构建成功且生成产物可启动，仍需在发布记录中保留该 warning 和人工启动结果。
- GitNexus 对 Wails 动态绑定、Go interface dispatch、TypeScript property access 的调用图只能提供下界；`UNKNOWN` 必须用源码搜索和定向测试补证，不能解释成“无调用方”。
- `projectwrite` 是进程内 Store 写入互斥；跨进程保护依赖 Core 使用的小说目录 book lease。若外部工具绕过 Core/Host 直接改文件，V1 不承诺事务级一致性。
- 项目视图刷新失败不会回滚已成功的 Core 操作：UI 应接受 Core 的 completed/Synced 事实并显示 warning；用户可通过重新打开项目刷新视图。
- 长篇项目当前使用折叠树与有界 Runtime 日志；1000+ 章仍应在目标 Windows 机器上做实际滚动、切章和 Review 复核，尚未承诺全树虚拟化。
- Import 只支持 Core 可解码的文本源（UTF-8 / GB18030）；EPUB、PDF、DOCX 不属于 V1 Import 语义。EPUB 仅用于 Export。
- V1 不包含 Fact Engine、Character Knowledge、Proposal、Dependency Graph、ContextManifest、Impact Analysis、Repair/Replan、云同步、多人协作或插件系统。
