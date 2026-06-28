package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/chunker"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/db"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/storage"
)

func main() {
	fmt.Println("🚀 Starting CDC Deduplication Engine Benchmark...")

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		if _, err := os.Stat("../bin"); err == nil {
			dataDir = "../data"
		} else {
			dataDir = "./data"
		}
	}
	os.MkdirAll(dataDir, 0755)

	dbPath := filepath.Join(dataDir, "dedup_index.db")
	objDir := filepath.Join(dataDir, "objects")

	database, err := db.OpenDB(dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()
	storeBackend := storage.NewLocalStorage(objDir)
	cdc := chunker.NewCDCChunker(16*1024, 64*1024, 256*1024)

	fileSize := int64(1024 * 1024 * 1024) // 1GB 测试大文件（题目要求）
	tempDir := filepath.Join(os.TempDir(), "cdc_bench_tmp")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	v1Path := filepath.Join(tempDir, "v1_original.dat")
	v2Path := filepath.Join(tempDir, "v2_modified.dat")

	fmt.Printf("1️⃣ Generating %s random original file (V1)...\n", humanize.Bytes(uint64(fileSize)))
	generateRandomFile(v1Path, fileSize)

	fmt.Println("2️⃣ Storing V1 into CDC Dedup Engine...")
	t0 := time.Now()
	store(v1Path, cdc, storeBackend, database)
	fmt.Printf("⏱️ V1 Stored in %v\n\n", time.Since(t0).Round(time.Millisecond))

	fmt.Println("3️⃣ Creating V2: Copying V1 and modifying only 100 bytes in the exact middle...")
	copyFileAndModify(v1Path, v2Path, fileSize/2, 100)

	fmt.Println("4️⃣ Storing V2 into CDC Dedup Engine...")
	t1 := time.Now()
	store(v2Path, cdc, storeBackend, database)
	fmt.Printf("⏱️ V2 Stored in %v\n\n", time.Since(t1).Round(time.Millisecond))

	fmt.Println("🏆 === BENCHMARK RESULTS & STORAGE EFFICIENCY ===")
	stats, _ := database.GetStats()
	logicalSize := stats["logical_size"].(int64)
	physicalSize := stats["physical_size"].(int64)

	fmt.Printf("📁 Total Logical Files Size : %s (2 versions of %s)\n", humanize.Bytes(uint64(logicalSize)), humanize.Bytes(uint64(fileSize)))
	fmt.Printf("💽 Actual Physical Storage  : %s\n", humanize.Bytes(uint64(physicalSize)))
	fmt.Printf("🎉 Global Dedup Ratio       : %v%%\n", stats["dedup_ratio_percent"])
	fmt.Printf("💡 Space Saved              : %s\n", humanize.Bytes(uint64(logicalSize-physicalSize)))
}

func generateRandomFile(path string, size int64) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("❌ Failed to create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	buf := make([]byte, 1024*1024)
	var written int64
	for written < size {
		rand.Read(buf)
		toWrite := int64(len(buf))
		if size-written < toWrite {
			toWrite = size - written
		}
		if _, err := f.Write(buf[:toWrite]); err != nil {
			fmt.Printf("❌ Write error: %v\n", err)
			os.Exit(1)
		}
		written += toWrite
	}
}

func copyFileAndModify(src, dst string, offset int64, modLen int) {
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("❌ Failed to read source file: %v\n", err)
		os.Exit(1)
	}
	mod := make([]byte, modLen)
	for i := 0; i < modLen; i++ {
		mod[i] = 0xFF // 修改为固定全 F
	}
	copy(data[offset:], mod)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		fmt.Printf("❌ Failed to write modified file: %v\n", err)
		os.Exit(1)
	}
}

func store(filePath string, cdc *chunker.CDCChunker, store storage.Backend, database *db.MetadataDB) {
	fileInfo, _ := os.Stat(filePath)
	f, _ := os.Open(filePath)
	defer f.Close()

	var chunkHashes []string
	rootHasher := sha256.New()
	newChunks := 0
	dedupChunks := 0

	cdc.ChunkStream(f, func(c chunker.Chunk) error {
		chunkHashes = append(chunkHashes, c.Hash)
		rootHasher.Write([]byte(c.Hash))
		isNew, _ := database.AddChunk(c.Hash, c.Size)
		if isNew {
			newChunks++
			store.Put(c.Hash, c.Data)
		} else {
			dedupChunks++
		}
		return nil
	})
	rootHash := hex.EncodeToString(rootHasher.Sum(nil))
	database.AddFile(rootHash, filepath.Base(filePath), fileInfo.Size(), chunkHashes)
	fmt.Printf("   -> [%s] RootHash: %s | Chunks: %d (New: %d, Dedup Reused: %d)\n", filepath.Base(filePath), rootHash[:12]+"...", len(chunkHashes), newChunks, dedupChunks)
}
