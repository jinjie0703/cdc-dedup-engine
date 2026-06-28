#!/usr/bin/env bash
set -e

echo "🚀 [macOS] 开始一键编译部署 CDC 增量去重引擎..."

# 自动定位到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

mkdir -p bin

echo "📦 正在编译 Go 后端引擎..."
cd engine
go build -o ../bin/cdc-dedup ./cmd/cdc-dedup
cd ..

echo "🎨 正在构建 React 前端应用..."
cd frontend
npm install
npm run build
cd ..

echo "⚙️ 正在后台启动 Go API 服务 (监听端口 8080)..."
nohup ./bin/cdc-dedup server --port 8080 > backend.log 2>&1 &
echo $! > backend.pid

echo "🌐 正在后台启动前端 Web 服务 (监听端口 3000)..."
cd frontend
nohup npm run preview -- --port 3000 > ../frontend.log 2>&1 &
echo $! > ../frontend.pid
cd ..

sleep 2
echo "🎉 正在打开浏览器访问看板页面..."
open http://localhost:3000

echo "✅ 部署完成！日志已记录至 backend.log 与 frontend.log"
