param(
    [string]$Name
)

# Set output encoding to UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Write-Host "[Windows PowerShell] Starting project cleanup and packaging..." -ForegroundColor Cyan

# Auto navigate to repository root directory
Set-Location -Path "$PSScriptRoot\.."

if (-not $Name) {
    $Name = Read-Host "Enter your name for the archive file (e.g. ZhangSan)"
    if (-not $Name) {
        $Name = "submission"
    }
}

Write-Host "`n1/3 Cleaning up temporary build artifacts and node_modules to reduce zip size..." -ForegroundColor Yellow
Stop-Process -Name "cdc-dedup" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "data" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "frontend\node_modules" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "frontend\dist" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "engine\data" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "engine\test_data" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "bin" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path "*.exe", "engine\*.exe", "*.log", "*.pid", "*.txt", "*.dat" -Force -ErrorAction SilentlyContinue

Write-Host "2/3 Collecting repository files (including .git history)..." -ForegroundColor Yellow
$ZipPath = Join-Path -Path (Split-Path -Parent $PWD) -ChildPath "$Name.zip"
if (Test-Path $ZipPath) {
    Remove-Item -Path $ZipPath -Force
}

$Items = Get-ChildItem -Path . -Force | Where-Object { $_.Name -ne "." -and $_.Name -ne ".." } | Select-Object -ExpandProperty FullName

Write-Host "3/3 Compressing files into $ZipPath..." -ForegroundColor Green
Compress-Archive -Path $Items -DestinationPath $ZipPath -Force

Write-Host "`nSuccessfully created archive: $ZipPath" -ForegroundColor Green
Write-Host "You can now send this archive file to Leming@whyfjz.com!" -ForegroundColor Cyan
