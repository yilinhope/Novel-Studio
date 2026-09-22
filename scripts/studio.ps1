param([ValidateSet('build','dev','test')][string]$Mode = 'build')
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$portable = Join-Path $root '.cache/toolchain/go/bin'
if (Test-Path (Join-Path $portable 'go.exe')) { $env:PATH = "$portable;$env:PATH" }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw '请安装 Go 1.25.5 或更高版本。' }
Push-Location $root
try {
    if ($Mode -eq 'test') {
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Go 测试失败' }
        Push-Location desktop/frontend
        try {
            npm ci
            if ($LASTEXITCODE -ne 0) { throw '前端依赖安装失败' }
            npm test
            if ($LASTEXITCODE -ne 0) { throw '前端测试失败' }
            npm run build
            if ($LASTEXITCODE -ne 0) { throw '前端构建失败' }
        } finally { Pop-Location }
    } else {
        Push-Location cmd/novel-studio
        try {
            go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 $Mode
            if ($LASTEXITCODE -ne 0) { throw 'Wails 构建或启动失败' }
        } finally { Pop-Location }
    }
} finally { Pop-Location }
