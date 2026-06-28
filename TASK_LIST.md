# CDC Dedup Engine 研发与工程部署任务清单 (TASK_LIST)

### 阶段一：架构设计与基础引擎 (已完成)

- [x] 初始化 Git 仓库，确定 Monorepo 工程结构 (`engine/` + `frontend/`)
- [x] 撰写项目设计文档 (`DESIGN.md`) 与 AI 规范约定 (`AI_SPEC.md`)
- [x] 实现基于 FastCDC (Gear Rolling Hash) 的可变分块算法 (`engine/pkg/chunker`)
- [x] 实现基于 `SHA-256` 指纹的内容寻址物理存储接口 (`engine/pkg/storage`)
- [x] 引入纯 Go 内嵌 SQLite (`modernc.org/sqlite`)，实现元数据与引用计数管理 (`engine/pkg/db`)
- [x] 构建强大的 CLI 交互命令行 (`store`, `fetch`, `stats`, `gc`)

### 阶段二：自动化基准与可视化看板 (已完成)

- [x] 编写一键验证测试脚本，自动生成 50MB 随机文件，验证中间修改 100 字节后的 99%+ 去重复用率 (`engine/cmd/benchmark`)
- [x] 基于 React + Vite + TypeScript 初始化前端架构
- [x] 引入最新版 Ant Design 组件库与暗黑科技美学主题 (`frontend/src/App.tsx`)
- [x] 实现前后端 RESTful API 契约对接与优雅降级 Mock 模式
- [x] 清理测试临时垃圾数据，配置 `.gitignore`，维持 Git 仓库极度轻量干净

### 阶段三：环境配置抽离与多平台跨操作系统部署 (已完成)

- [x] 抽离前后端配置参数至 `.env` 与 `.env.example`（遵循高内聚低耦合规范）
- [x] 编写 Windows 系统一键部署启动脚本 (`deploy.bat` / `deploy.ps1`)
- [x] 编写 Linux 系统一键部署启动脚本 (`deploy-linux.sh`)
- [x] 编写 macOS 系统一键部署启动脚本 (`deploy-mac.sh`)
- [x] 验证部署流程，更新 Git 仓库提交
