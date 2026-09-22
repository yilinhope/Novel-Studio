# 外部来源与实现约束

## ainovel-cli

Repository:
https://github.com/voocel/ainovel-cli

V1 以实际克隆到工作区的源码为唯一代码事实来源。  
设计文档中出现的 package 名称是架构意图，不应覆盖当前源码事实。

开发开始时请记录：

```bash
git rev-parse HEAD
go version
go test ./...
```

并将结果写入项目开发日志。

## Wails

使用当前项目环境可稳定安装的 Wails 版本。  
初始化后请提交 Wails 配置、前端依赖锁文件以及构建说明。

## AI-Novel-Writer

Repository:
https://github.com/EthanYoQ/AI-Novel-Writer

V1 只吸收过产品思想背景。  
**不要复制其 GPL 桌面源码到 Novel Studio V1。**

## Stitch

Stitch 导出的 HTML/CSS/截图是设计参考，不是生产代码架构。

生产实现必须：

```text
React + TypeScript
Go/Wails Bridge
真实 Core ViewModel
```

不要让 UI mock 反过来定义 ainovel Core。
