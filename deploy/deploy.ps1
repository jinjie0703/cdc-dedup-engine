# Set output encoding to UTF8 to prevent display issues
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Write-Host "[Windows PowerShell] Starting CDC Dedup Engine deployment..." -ForegroundColor Cyan

# Auto navigate to repository root directory
Set-Location -Path "$PSScriptRoot\.."

if (-not (Test-Path "./bin")) { New-Item -ItemType Directory -Force -Path "./bin" | Out-Null }

Write-Host "1/4 Building Go Backend Engine..." -ForegroundColor Yellow
Push-Location engine
go build -o ../bin/cdc-dedup.exe ./cmd/cdc-dedup
Pop-Location

Write-Host "2/4 Building React Frontend..." -ForegroundColor Yellow
Push-Location frontend
npm install
npm run build
Pop-Location

Write-Host "3/4 Starting Go API Server on port 8080..." -ForegroundColor Green
Start-Process -FilePath "./bin/cdc-dedup.exe" -ArgumentList "server", "--port", "8080" -NoNewWindow

Write-Host "4/4 Starting Frontend Web Server on port 3000..." -ForegroundColor Green
Push-Location frontend
Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "npm", "run", "preview", "--", "--port", "3000" -NoNewWindow
Pop-Location

Start-Sleep -Seconds 2
Write-Host "Opening Browser at http://localhost:3000..." -ForegroundColor Cyan
Start-Process "http://localhost:3000"

Write-Host "Deployment completed successfully!" -ForegroundColor Green
