# CDC Dedup Engine (基于 Go 的内容可变分块去重引擎)

**CDC Dedup Engine** 是一个面向大文件的分布式去重存储引擎。基于 **Go 语言** 与 **FastCDC 可变分块算法** 开发，通过对大文件进行内容寻址切割（Content-Defined Chunking），有效解决局部修改导致整体哈希失效的问题，实现高达 99%+ 的极速增量存储去重！配套现代化的 React + Vite 动态可视化看板。

## 🌟 核心特性

- **🚀 极速 FastCDC 分块**：基于预生成 Gear 哈希映射表的滑动窗口算法，极低 CPU 消耗，吞吐量极高。
- **📦 内容寻址存储 (CAS)**：数据块以 `SHA-256` 指纹进行寻址，实现天然全局去重。
- **🗄️ 纯 Go 内嵌 SQLite**：采用 `modernc.org/sqlite` 驱动，零 CGO 依赖，跨平台一键编译运行！
- **☁️ 抽象云端存储扩展**：内置本地两级目录 CAS 引擎，且已完成对阿里云 OSS 等云端对象存储的接口支持。
- **🎨 现代化可视化看板**：基于 React + Vite + Ant Design 的科技蓝暗黑主题看板，实时监控去重率与存储指纹。
- **🤖 定制化智能研发体系**：配备收纳于 `.agents/skills` 的高级技术规范，保障架构先行与测试驱动。

---

## 📁 工程目录架构

```text
cdc-dedup-engine/
├── DESIGN.md               # 📖 系统核心架构设计与切块数学原理文档 (重点审查)
├── README.md               # 📖 项目操作指南
├── docs/                   # 📑 规范与任务收纳目录
│   ├── TASK_LIST.md        # 里程碑任务分解与全量验收清单
│   └── AI_SPEC.md          # 智能体协同研发规范约定
├── deploy/                 # 🚀 跨平台一键自动化构建部署脚本
│   ├── deploy.bat          # Windows 批处理启动脚本
│   ├── deploy.ps1          # Windows PowerShell 启动脚本
│   ├── deploy-linux.sh     # Linux 一键编译启动脚本
│   └── deploy-mac.sh       # macOS 一键编译启动脚本
├── engine/                 # ⚙️ Go 后端去重引擎模块
│   ├── cmd/                # CLI 与服务入口 (cdc-dedup / benchmark)
│   └── pkg/                # 核心算法包 (chunker切块 / storage存储 / db元数据 / api服务)
└── frontend/               # 🎨 React + Vite + TypeScript 可视化看板应用
```

---

## 🛠️ 快速开始与一键部署

我们推荐使用 `deploy/` 目录下的自动化脚本进行跨平台一键编译与服务启动（会自动启动 8080 后端 API 与 3000 前端看板页面）：

### Windows 环境

在 PowerShell 中运行：

```powershell
.\deploy\deploy.ps1
# 或通过 cmd 运行：.\deploy\deploy.bat
```

### Linux / macOS 环境

在终端中运行：

```bash
# Linux
bash deploy/deploy-linux.sh

# macOS
bash deploy/deploy-mac.sh
```

---

## 💻 命令行操作指南 (CLI)

如果你希望单独使用 Go 后端引擎命令行：

```bash
# 1. 编译生成二进制执行文件
go build -o bin/cdc-dedup.exe ./engine/cmd/cdc-dedup

# 2. 存储文件并返回根哈希 (Root Hash)
./bin/cdc-dedup.exe store ./sample_big_file.dat

# 3. 查看文件分块统计与全局去重率
./bin/cdc-dedup.exe stats ./sample_big_file.dat

# 4. 根据根哈希无损还原完整文件
./bin/cdc-dedup.exe fetch <root-hash> ./output_restored.dat

# 5. 执行垃圾回收 (清理无引用数据块)
./bin/cdc-dedup.exe gc
```

### 一键基准自动化测试 (Benchmark)

验证当生成大文件并从中修改 100 字节时，系统依旧能保持 99%+ 的去复用率：

```bash
go run ./engine/cmd/benchmark
```

详细的设计理论与算法推导公式，请参阅根目录下的 [DESIGN.md](./DESIGN.md)。
