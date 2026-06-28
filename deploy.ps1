Write-Host "🚀 [Windows PowerShell] 开始一键编译部署 CDC 增量去重引擎..." -ForegroundColor Cyan

if (-not (Test-Path "./bin")) { New-Item -ItemType Directory -Force -Path "./bin" | Out-Null }

Write-Host "📦 正在编译 Go 后端引擎..." -ForegroundColor Yellow
Push-Location engine
go build -o ../bin/cdc-dedup.exe ./cmd/cdc-dedup
Pop-Location

Write-Host "🎨 正在构建 React 前端应用..." -ForegroundColor Yellow
Push-Location frontend
npm install
npm run build
Pop-Location

Write-Host "⚙️ 正在启动 Go API 服务 (端口 8080)..." -ForegroundColor Green
Start-Process -FilePath "./bin/cdc-dedup.exe" -ArgumentList "server", "--port", "8080" -NoNewWindow

Write-Host "🌐 正在启动前端 Web 服务 (端口 3000)..." -ForegroundColor Green
Push-Location frontend
Start-Process -FilePath "npm" -ArgumentList "run", "preview", "--", "--port", "3000" -NoNewWindow
Pop-Location

Start-Sleep -Seconds 2
Write-Host "🎉 正在打开浏览器..." -ForegroundColor Cyan
Start-Process "http://localhost:3000"

Write-Host "✅ 部署启动完成！" -ForegroundColor Green
