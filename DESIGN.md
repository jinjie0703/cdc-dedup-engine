# 系统架构与详细设计文档 (DESIGN.md)

本设计文档详细阐述了 **CDC Dedup Engine** 的分块策略、数学原理、底层存储抽象以及数据恢复流程。

---

## 1. CDC 可变分块策略与数学原理

传统云盘在处理文件上传时，如果采用固定分块（如按 64KB 切割），一旦用户在文件头部插入 1 个字节，随后所有的 64KB 分块边界都会向后偏移 1 字节，导致所有后续分块的哈希值发生改变，去重率骤降为 0。

### 1.1 滚动哈希 (Rolling Hash) 与分块边界

本引擎采用 **可变分块 (Content-Defined Chunking, CDC)** 策略，核心依赖 **拉宾指纹 (Rabin Fingerprint)** 或类简化滚动哈希滑动窗口。

当读取文件流时，维护一个大小为 $W$（例如 48 字节）的滑动窗口。在每个字节偏移处计算窗口内的滚动哈希值 $H$：
$$H = (H_{prev} \times P + C_{new} - C_{old} \times P^W) \pmod M$$

当且仅当哈希值满足特定的掩码条件（例如低位若干 bit 为 0）：
$$H \mathbin{\&} \text{MASK} == 0$$
系统即认定此处为一个分块的物理边界（Chunk Boundary）。

### 1.2 大小边界约束 (Min/Max Constraints)

为了防止极端情况（如完全相同的重复字符导致块极小，或随机数据导致块极大）：

- **最小块大小 (Min Size)**：设定为 `16 KB`。窗口在未达到最小限制前不进行边界判定，防止产生大量碎块导致元数据膨胀。
- **最大块大小 (Max Size)**：设定为 `512 KB`。当块大小达到此上限仍未触发掩码条件时，强制进行切块，防止单块占用过多内存。
- **目标平均块大小 (Avg Size)**：设定为 `64 KB` ~ `128 KB`，对应的掩码为 `0xFFFF`。

---

## 2. 哈希表结构与 SQLite 元数据设计

系统采用 **内容寻址存储 (Content-Addressable Storage, CAS)**，分块本身不记录文件名和偏移量，而是只依靠内容的 `SHA-256` 哈希指纹来寻址。

我们在本地 SQLite 数据库中维护三个核心关系表：

### 2.1 数据表定义

```sql
-- 1. 文件元数据表：记录整个文件的根哈希（Tree Hash）与基本信息
CREATE TABLE IF NOT EXISTS files (
    root_hash TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. 数据块全局索引表：记录全局所有唯一的物理数据块及引用计数
CREATE TABLE IF NOT EXISTS chunks (
    chunk_hash TEXT PRIMARY KEY,
    size INTEGER NOT NULL,
    ref_count INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. 文件-数据块映射表：记录某个文件由哪些有序的数据块组成
CREATE TABLE IF NOT EXISTS file_chunks (
    root_hash TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_hash TEXT NOT NULL,
    PRIMARY KEY (root_hash, chunk_index),
    FOREIGN KEY (root_hash) REFERENCES files(root_hash) ON DELETE CASCADE,
    FOREIGN KEY (chunk_hash) REFERENCES chunks(chunk_hash)
);
```

---

## 3. 存储后端抽象 (Storage Backend)

系统解耦了“分块逻辑”与“物理存储介质”，通过 `StorageBackend` 接口实现插件化驱动：

- **LocalStorageBackend (默认)**：将数据块存储在本地 `./data/objects/` 目录下。采用两级目录哈希打散策略（如 `ab/cd/abcd1234...`），避免单目录下文件过多导致操作系统的 inode 检索性能下降。
- **CloudStorageBackend (云端扩展)**：兼容腾讯云 COS、阿里云 OSS 或 AWS S3。数据块存为对象桶中的 key，实现远端对象存储原生去重。

---

## 4. 核心执行流程

### 4.1 存储流程 (`store`)

1. 打开待存储文件读取流。
2. 运行 CDC 滚动哈希切块器，动态切割出有序的数据块列表。
3. 对每个分块计算 `SHA-256` 哈希。
4. 查询数据库 `chunks` 表：
   - 若不存在：将分块物理写入 Storage Backend，并在 `chunks` 插入新记录 (`ref_count = 1`)。
   - 若已存在：跳过物理写入（实现去重！），并在 `chunks` 中将其 `ref_count += 1`。
5. 将所有分块的哈希拼接后二次计算 `SHA-256` 作为整个文件的 `root_hash`。
6. 写入 `files` 与 `file_chunks` 表映射关系。

### 4.2 还原流程 (`fetch`)

1. 根据用户提供的 `root_hash` 查询 `file_chunks` 表，按 `chunk_index` 升序获取所有分块的 `chunk_hash` 列表。
2. 依次从 Storage Backend 提取对应哈希的数据块内容。
3. 按顺序追加写入到目标文件路径，完成 100% 字节无损还原。

### 4.3 垃圾回收流程 (`gc`)

1. 扫描 `chunks` 表中 `ref_count <= 0` 的废弃数据块。
2. 调用 Storage Backend 删除物理存储中的对应数据文件。
3. 从 `chunks` 表中物理清除该元数据记录。
