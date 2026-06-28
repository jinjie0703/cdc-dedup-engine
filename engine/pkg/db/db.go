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
		PRIMARY KEY (root_hash, chunk_index)
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

	_, err = tx.Exec("INSERT OR REPLACE INTO files (root_hash, file_name, file_size) VALUES (?, ?, ?)", rootHash, fileName, fileSize)
	if err != nil {
		return err
	}

	// 清除旧的映射（如果存在相同 rootHash 重复存储的情况）
	_, err = tx.Exec("DELETE FROM file_chunks WHERE root_hash = ?", rootHash)
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
		f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", tStr)
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
