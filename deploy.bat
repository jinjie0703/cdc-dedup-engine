@echo off
chcp 65001 > nul
echo 🚀 [Windows] 开始一键编译部署 CDC 增量去重引擎...

if not exist bin mkdir bin

echo 📦 正在编译 Go 后端引擎...
cd engine
go build -o ../bin/cdc-dedup.exe ./cmd/cdc-dedup
cd ..

echo 🎨 正在构建 React 前端应用...
cd frontend
call npm install
call npm run build
cd ..

echo ⚙️ 正在后台启动 Go API 服务 (端口 8080)...
start /B bin\cdc-dedup.exe server --port 8080

echo 🌐 正在启动前端 Web 服务 (端口 3000)...
cd frontend
start /B npm run preview -- --port 3000
cd ..

timeout /t 2 /nobreak > nul
echo 🎉 正在自动打开浏览器访问看板页面...
start http://localhost:3000

echo ✅ 部署部署完成！请在浏览器中体验 CDC 去重引擎！
