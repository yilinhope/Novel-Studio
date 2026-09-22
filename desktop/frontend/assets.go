package frontend

import "embed"

// Assets 包含前端构建产物；构建桌面程序前执行 npm ci 和 npm run build。
//
//go:embed all:dist
var Assets embed.FS
