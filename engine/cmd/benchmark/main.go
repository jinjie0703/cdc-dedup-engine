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

	testDir := "./test_data"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)

	dbPath := filepath.Join(testDir, "bench_index.db")
	objDir := filepath.Join(testDir, "objects")

	database, _ := db.OpenDB(dbPath)
	defer database.Close()
	storeBackend := storage.NewLocalStorage(objDir)
	cdc := chunker.NewCDCChunker(16*1024, 64*1024, 256*1024)

	fileSize := int64(50 * 1024 * 1024) // 50MB 测试大文件
	v1Path := filepath.Join(testDir, "v1_original.dat")
	v2Path := filepath.Join(testDir, "v2_modified.dat")

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
	f, _ := os.Create(path)
	defer f.Close()
	buf := make([]byte, 1024*1024)
	var written int64
	for written < size {
		rand.Read(buf)
		toWrite := int64(len(buf))
		if size-written < toWrite {
			toWrite = size - written
		}
		f.Write(buf[:toWrite])
		written += toWrite
	}
}

func copyFileAndModify(src, dst string, offset int64, modLen int) {
	data, _ := os.ReadFile(src)
	mod := make([]byte, modLen)
	for i := 0; i < modLen; i++ {
		mod[i] = 0xFF // 修改为固定全 F
	}
	copy(data[offset:], mod)
	os.WriteFile(dst, data, 0644)
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
