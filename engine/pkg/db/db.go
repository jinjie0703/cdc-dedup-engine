package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type MetadataDB struct {
	db *sql.DB
}

type FileMeta struct {
	RootHash  string    `json:"root_hash"`
	FileName  string    `json:"file_name"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

type ChunkMeta struct {
	ChunkHash string `json:"chunk_hash"`
	Size      int    `json:"size"`
	RefCount  int    `json:"ref_count"`
}

func OpenDB(dbPath string) (*MetadataDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 启用 SQLite 外键约束
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	m := &MetadataDB{db: database}
	if err := m.initTables(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MetadataDB) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		root_hash TEXT PRIMARY KEY,
		file_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chunks (
		chunk_hash TEXT PRIMARY KEY,
		size INTEGER NOT NULL,
		ref_count INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS file_chunks (
		root_hash TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		chunk_hash TEXT NOT NULL,
		PRIMARY KEY (root_hash, chunk_index),
		FOREIGN KEY (root_hash) REFERENCES files(root_hash) ON DELETE CASCADE,
		FOREIGN KEY (chunk_hash) REFERENCES chunks(chunk_hash)
	);
	`
	_, err := m.db.Exec(schema)
	return err
}

// Close 关闭数据库连接
func (m *MetadataDB) Close() error {
	return m.db.Close()
}

// AddChunk 插入分块，若已存在则增加 ref_count
func (m *MetadataDB) AddChunk(hash string, size int) (bool, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var existingRefCount int
	err = tx.QueryRow("SELECT ref_count FROM chunks WHERE chunk_hash = ?", hash).Scan(&existingRefCount)
	if err == sql.ErrNoRows {
		_, err = tx.Exec("INSERT INTO chunks (chunk_hash, size, ref_count) VALUES (?, ?, 1)", hash, size)
		if err != nil {
			return false, err
		}
		tx.Commit()
		return true, nil // 物理新增了数据块
	} else if err != nil {
		return false, err
	}

	_, err = tx.Exec("UPDATE chunks SET ref_count = ref_count + 1 WHERE chunk_hash = ?", hash)
	if err != nil {
		return false, err
	}
	tx.Commit()
	return false, nil // 复用了已有数据块
}

// AddFile 记录完整文件及其切块序列
func (m *MetadataDB) AddFile(rootHash, fileName string, fileSize int64, chunkHashes []string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 如果该 rootHash 已存在，先对旧的 chunk 引用做 ref_count -= 1
	oldChunks, _ := m.getFileChunksTx(tx, rootHash)
	if len(oldChunks) > 0 {
		for _, ch := range oldChunks {
			if _, err := tx.Exec("UPDATE chunks SET ref_count = ref_count - 1 WHERE chunk_hash = ?", ch); err != nil {
				return err
			}
		}
		if _, err := tx.Exec("DELETE FROM file_chunks WHERE root_hash = ?", rootHash); err != nil {
			return err
		}
	}

	_, err = tx.Exec("INSERT OR REPLACE INTO files (root_hash, file_name, file_size) VALUES (?, ?, ?)", rootHash, fileName, fileSize)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO file_chunks (root_hash, chunk_index, chunk_hash) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for idx, ch := range chunkHashes {
		if _, err := stmt.Exec(rootHash, idx, ch); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// getFileChunksTx 在事务内获取文件的 chunk 列表
func (m *MetadataDB) getFileChunksTx(tx *sql.Tx, rootHash string) ([]string, error) {
	rows, err := tx.Query("SELECT chunk_hash FROM file_chunks WHERE root_hash = ? ORDER BY chunk_index ASC", rootHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		chunks = append(chunks, h)
	}
	return chunks, nil
}

// GetFileChunks 按照 index 升序获取指定文件的所有数据块哈希
func (m *MetadataDB) GetFileChunks(rootHash string) ([]string, error) {
	rows, err := m.db.Query("SELECT chunk_hash FROM file_chunks WHERE root_hash = ? ORDER BY chunk_index ASC", rootHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		chunks = append(chunks, h)
	}
	return chunks, nil
}

// GetStats 获取全局统计数据
func (m *MetadataDB) GetStats() (map[string]interface{}, error) {
	var totalFiles int
	var logicalSize int64
	var totalChunks int
	var physicalSize int64

	m.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM files").Scan(&totalFiles, &logicalSize)
	m.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(size), 0) FROM chunks").Scan(&totalChunks, &physicalSize)

	var totalChunkRefs int
	m.db.QueryRow("SELECT COUNT(*) FROM file_chunks").Scan(&totalChunkRefs)

	dedupRatio := 0.0
	if logicalSize > 0 {
		saved := logicalSize - physicalSize
		if saved > 0 {
			dedupRatio = float64(saved) / float64(logicalSize) * 100.0
		}
	}

	return map[string]interface{}{
		"total_files":         totalFiles,
		"total_chunks":        totalChunkRefs,
		"unique_chunks":       totalChunks,
		"logical_size":        logicalSize,
		"physical_size":       physicalSize,
		"dedup_ratio_percent": fmt.Sprintf("%.2f", dedupRatio),
	}, nil
}

// ListFiles 获取所有文件列表
func (m *MetadataDB) ListFiles() ([]FileMeta, error) {
	rows, err := m.db.Query("SELECT root_hash, file_name, file_size, created_at FROM files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileMeta
	for rows.Next() {
		var f FileMeta
		var tStr string
		if err := rows.Scan(&f.RootHash, &f.FileName, &f.FileSize, &tStr); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, tStr); err == nil {
			f.CreatedAt = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", tStr); err == nil {
			f.CreatedAt = t
		} else {
			f.CreatedAt = time.Now()
		}
		files = append(files, f)
	}
	return files, nil
}

// RunGC 清理引用计数小于等于 0 的数据块
func (m *MetadataDB) RunGC(onDelete func(hash string) error) (int, error) {
	rows, err := m.db.Query("SELECT chunk_hash FROM chunks WHERE ref_count <= 0")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return 0, err
		}
		toDelete = append(toDelete, h)
	}

	deletedCount := 0
	for _, h := range toDelete {
		if err := onDelete(h); err == nil {
			m.db.Exec("DELETE FROM chunks WHERE chunk_hash = ?", h)
			deletedCount++
		}
	}
	return deletedCount, nil
}

// GetFileStatsByName 根据文件名查询该文件的分块信息与去重率
func (m *MetadataDB) GetFileStatsByName(fileName string) (map[string]interface{}, error) {
	var rootHash string
	var fileSize int64
	err := m.db.QueryRow("SELECT root_hash, file_size FROM files WHERE file_name = ? ORDER BY created_at DESC LIMIT 1", fileName).Scan(&rootHash, &fileSize)
	if err != nil {
		return nil, fmt.Errorf("file '%s' not found in database", fileName)
	}

	chunks, err := m.GetFileChunks(rootHash)
	if err != nil {
		return nil, err
	}

	// 统计该文件中有多少块是全局唯一的 vs 被复用的
	uniqueInFile := make(map[string]bool)
	totalChunkSize := int64(0)
	for _, ch := range chunks {
		uniqueInFile[ch] = true
		var size int
		m.db.QueryRow("SELECT size FROM chunks WHERE chunk_hash = ?", ch).Scan(&size)
		totalChunkSize += int64(size)
	}

	// 全局去重率
	globalStats, _ := m.GetStats()

	return map[string]interface{}{
		"file_name":           fileName,
		"root_hash":           rootHash,
		"file_size":           fileSize,
		"total_chunks":        len(chunks),
		"unique_chunks":       len(uniqueInFile),
		"global_dedup_ratio":  globalStats["dedup_ratio_percent"],
	}, nil
}
