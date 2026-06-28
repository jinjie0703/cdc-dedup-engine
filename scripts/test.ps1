# Set output encoding to UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Write-Host "[Windows PowerShell] Running CDC Dedup Engine Automated Tests..." -ForegroundColor Cyan

# Auto navigate to repository root directory
Set-Location -Path "$PSScriptRoot\.."

Write-Host "1/2 Running Go Unit Tests..." -ForegroundColor Yellow
Push-Location engine
go test -v ./...

Write-Host "`n2/2 Running CDC Benchmark & Deduplication Verification..." -ForegroundColor Yellow
go run ./cmd/benchmark
Pop-Location

Write-Host "`nAll tests completed successfully!" -ForegroundColor Green
