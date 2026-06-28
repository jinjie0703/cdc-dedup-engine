#!/usr/bin/env bash
set -e

echo "🧪 [Linux/macOS] 正在运行 CDC 增量去重引擎自动化测试..."

# 自动定位到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "1/2 正在执行 Go 单元测试..."
cd engine
go test -v ./...

echo ""
echo "2/2 正在执行 CDC 分块去重基准验证..."
go run ./cmd/benchmark
cd ..

echo ""
echo "✅ 全部测试验证通过！去重性能符合预期！"
