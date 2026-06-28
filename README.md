# CDC Dedup Engine (基于 Go 的内容可变分块去重引擎)

**CDC Dedup Engine** 是一个面向大文件的分布式去重存储引擎。基于 **Go 语言** 与 **FastCDC 可变分块算法** 开发，通过对大文件进行内容寻址切割（Content-Defined Chunking），有效解决局部修改导致整体哈希失效的问题，实现高达 99%+ 的极速增量存储去重！

## 🌟 核心特性

- **🚀 极速 FastCDC 分块**：基于预生成 Gear 哈希映射表的滑动窗口算法，极低 CPU 消耗，吞吐量极高。
- **📦 内容寻址存储 (CAS)**：数据块以 `SHA-256` 指纹进行寻址，实现天然全局去重。
- **🗄️ 纯 Go 内嵌 SQLite**：采用 `modernc.org/sqlite` 驱动，零 CGO 依赖，跨平台一键编译运行！
- **☁️ 抽象存储驱动支持**：内置本地两级目录 CAS 引擎，同时抽象了 `StorageBackend` 接口，可无缝对接腾讯云 COS / 阿里云 OSS / S3。
- **🧹 自动垃圾回收 (GC)**：内置基于引用计数的孤立数据块清理能力。

---

## 🛠️ 快速开始

### 1. 编译项目

在项目根目录下直接编译生成二进制执行文件：

```bash
go build -o cdc-dedup.exe ./cmd/cdc-dedup
```

### 2. 命令行操作 (CLI)

```bash
# 1. 存储文件并返回根哈希 (Root Hash)
./cdc-dedup.exe store ./sample_big_file.dat

# 2. 查看文件分块统计与全局去重率
./cdc-dedup.exe stats ./sample_big_file.dat

# 3. 根据根哈希还原完整文件
./cdc-dedup.exe fetch <root-hash> ./output_restored.dat

# 4. 执行垃圾回收 (清理无引用数据块)
./cdc-dedup.exe gc
```

### 3. 一键运行基准自动化测试

我们提供了一个自动验证测试工具，动态生成 100MB 测试文件，修改局部 100 字节后存入第二个版本，自动统计与比对存储空间节省比率：

```bash
go test -v ./test/benchmark_test.go
# 或者直接运行自动化测试程序
go run ./cmd/benchmark
```

详细的设计文档与理论公式请参阅 [DESIGN.md](./DESIGN.md)。
