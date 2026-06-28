@echo off
chcp 65001 > nul
echo [Windows Batch] Running CDC Dedup Engine Automated Tests...

:: Auto navigate to repository root directory
cd /d "%~dp0\.."

echo 1/2 Running Go Unit Tests...
cd engine
go test -v ./...

echo.
echo 2/2 Running CDC Benchmark & Deduplication Verification...
go run ./cmd/benchmark
cd ..

echo.
echo All tests completed successfully!
pause
