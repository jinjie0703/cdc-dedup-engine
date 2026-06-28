@echo off
echo [Windows Batch] Starting CDC Dedup Engine deployment...

:: 自动切回项目根目录
cd /d "%~dp0\.."

if not exist bin mkdir bin

echo 1/4 Building Go Backend Engine...
cd engine
go build -o ../bin/cdc-dedup.exe ./cmd/cdc-dedup
cd ..

echo 2/4 Building React Frontend...
cd frontend
call npm install
call npm run build
cd ..

echo 3/4 Starting Go API Server on port 8080...
start /B bin\cdc-dedup.exe server --port 8080

echo 4/4 Starting Frontend Web Server on port 3000...
cd frontend
start /B npm run preview -- --port 3000
cd ..

timeout /t 2 /nobreak > nul
echo Opening Browser at http://localhost:3000...
start http://localhost:3000

echo Deployment completed successfully!
