# AI Development Specifications & Project Standards (AI_SPEC.md)

本文档沉淀了 **CDC Dedup Engine** 的全栈工程架构与研发规范。所有参与本仓库开发的 AI 或开发者请严格遵循以下准则。

---

## 1. 仓库整体目录结构

```text
cdc-dedup-engine/
├── AI_SPEC.md             # AI 研发规范与架构约定 (本文档)
├── README.md              # 项目总体说明与快速启动指南
├── DESIGN.md              # 核心 CDC 分块算法与 CAS 理论设计文档
├── .gitignore             # 全局 Git 忽略规则
├── engine/                # 🚀 后端：Go 去重存储引擎
│   ├── cmd/
│   │   └── cdc-dedup/     # CLI 与 API 服务入口 (`main.go`)
│   ├── pkg/
│   │   ├── chunker/       # FastCDC 滚动哈希切块核心逻辑
│   │   ├── storage/       # CAS 物理存储接口 (本地 & 远端 COS)
│   │   ├── db/            # 纯 Go 内嵌 SQLite 索引元数据操作
│   │   └── api/           # HTTP RESTful API 服务 (供前端调用)
│   ├── test/              # 自动化基准测试与单元测试
│   ├── go.mod
│   └── go.sum
└── frontend/              # 🎨 前端：React + Vite + TS + Ant Design
    ├── src/
    │   ├── components/    # 可复用 UI 组件 (去重环形图、分块列表等)
    │   ├── pages/         # 核心视图页面
    │   ├── services/      # HTTP API 请求封装
    │   ├── types/         # TypeScript 数据类型定义
    │   ├── App.tsx
    │   └── main.tsx
    ├── package.json
    └── vite.config.ts
```

---

## 2. 后端开发规范 (Go Engine)

1. **版本与依赖**：基于 Go 1.26+。数据库操作使用 `modernc.org/sqlite`（纯 Go 实现，**严禁引入任何 CGO 依赖**，确保 Windows 跨平台零编译错误）。
2. **包职责分解**：
   - `pkg/chunker`：只负责字节流分析与 FastCDC 边界计算，不涉及网络或磁盘保存。
   - `pkg/storage`：只负责按 `SHA-256` 哈希读写物理文件（本地两级打散或云端 COS），实现 `StorageBackend` 接口。
   - `pkg/db`：只负责维护 `files`, `chunks`, `file_chunks` 表的 CRUD 操作及事务。
   - `pkg/api`：启动 HTTP 服务（默认端口 `:8080`），提供 CORS 跨域支持，允许 Web 看板调用。
3. **命令行操作模式**：
   - `cdc-dedup store <filepath>`
   - `cdc-dedup fetch <root-hash> <outpath>`
   - `cdc-dedup stats <filepath/root-hash>`
   - `cdc-dedup gc`
   - `cdc-dedup server --port 8080` (启动 API 后端服务)

---

## 3. 前端开发规范 (React + Vite + TS + Antd)

1. **技术栈**：React 18+ / 19, Vite, TypeScript, Ant Design (`antd` 最新版), `@ant-design/icons`。
2. **样式风格**：遵循现代 Web 设计美学（科技蓝主题、玻璃拟态卡片、极简留白、平滑过渡动画）。严禁使用过于陈旧或简陋的默认排版。
3. **接口通信**：默认请求后端 API `http://localhost:8080/api/`。

---

## 4. 前后端 API 契约 (RESTful API Contract)

### 4.1 获取系统统计数据
- **GET** `/api/stats`
- **Response**:
  ```json
  {
    "code": 0,
    "data": {
      "total_files": 12,
      "total_chunks": 450,
      "unique_chunks": 120,
      "logical_size": 1048576000,
      "physical_size": 15728640,
      "dedup_ratio_percent": 98.5
    }
  }
  ```

### 4.2 获取文件列表
- **GET** `/api/files`
- **Response**:
  ```json
  {
    "code": 0,
    "data": [
      {
        "root_hash": "a1b2c3d4...",
        "file_name": "sample_v1.dat",
        "file_size": 104857600,
        "chunk_count": 85,
        "created_at": "2026-06-28 10:00:00"
      }
    ]
  }
  ```

### 4.3 触发垃圾回收
- **POST** `/api/gc`
