package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/api"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/chunker"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/db"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/storage"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
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

	subcmd := os.Args[1]
	switch subcmd {
	case "store":
		storeCmd := flag.NewFlagSet("store", flag.ExitOnError)
		storeCmd.Parse(os.Args[2:])
		if storeCmd.NArg() < 1 {
			fmt.Println("Usage: cdc-dedup store <file-path>")
			os.Exit(1)
		}
		filePath := storeCmd.Arg(0)
		storeFile(filePath, cdc, storeBackend, database)

	case "fetch":
		fetchCmd := flag.NewFlagSet("fetch", flag.ExitOnError)
		fetchCmd.Parse(os.Args[2:])
		if fetchCmd.NArg() < 2 {
			fmt.Println("Usage: cdc-dedup fetch <root-hash> <output-path>")
			os.Exit(1)
		}
		fetchFile(fetchCmd.Arg(0), fetchCmd.Arg(1), storeBackend, database)

	case "stats":
		statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
		statsCmd.Parse(os.Args[2:])
		if statsCmd.NArg() > 0 {
			showFileStats(statsCmd.Arg(0), database)
		} else {
			showStats(database)
		}

	case "gc":
		runGC(storeBackend, database)

	case "server":
		defaultPort := 8080
		if pStr := os.Getenv("PORT"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil {
				defaultPort = p
			}
		}
		serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
		port := serverCmd.Int("port", defaultPort, "HTTP API server port")
		serverCmd.Parse(os.Args[2:])
		srv := api.NewServer(database, storeBackend, *port)
		if err := srv.Start(); err != nil {
			fmt.Printf("❌ Server failed: %v\n", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CDC Dedup Engine CLI")
	fmt.Println("Usage:")
	fmt.Println("  cdc-dedup store <file-path>")
	fmt.Println("  cdc-dedup fetch <root-hash> <output-path>")
	fmt.Println("  cdc-dedup stats [file-path]")
	fmt.Println("  cdc-dedup gc")
	fmt.Println("  cdc-dedup server [--port 8080]")
}

func storeFile(filePath string, cdc *chunker.CDCChunker, store storage.Backend, database *db.MetadataDB) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("❌ File error: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("❌ Open error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Printf("📦 Storing file: %s (%s)...\n", filePath, humanize.Bytes(uint64(fileInfo.Size())))

	var chunkHashes []string
	rootHasher := sha256.New()
	newChunks := 0
	dedupChunks := 0

	err = cdc.ChunkStream(f, func(c chunker.Chunk) error {
		chunkHashes = append(chunkHashes, c.Hash)
		rootHasher.Write([]byte(c.Hash))

		isNew, err := database.AddChunk(c.Hash, c.Size)
		if err != nil {
			return err
		}
		if isNew {
			newChunks++
			return store.Put(c.Hash, c.Data)
		}
		dedupChunks++
		return nil
	})

	if err != nil {
		fmt.Printf("❌ Chunking error: %v\n", err)
		os.Exit(1)
	}

	rootHash := hex.EncodeToString(rootHasher.Sum(nil))
	if err := database.AddFile(rootHash, filepath.Base(filePath), fileInfo.Size(), chunkHashes); err != nil {
		fmt.Printf("❌ Metadata error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Success! Root Hash: %s\n", rootHash)
	fmt.Printf("📊 Chunks: %d Total (%d New written, %d Deduplicated reused)\n", len(chunkHashes), newChunks, dedupChunks)
}

func fetchFile(rootHash, outPath string, store storage.Backend, database *db.MetadataDB) {
	chunks, err := database.GetFileChunks(rootHash)
	if err != nil || len(chunks) == 0 {
		fmt.Printf("❌ Root hash not found or empty: %s\n", rootHash)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Printf("❌ Output dir error: %v\n", err)
		os.Exit(1)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("❌ Create file error: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	fmt.Printf("🔍 Restoring file from Root Hash: %s (%d chunks)...\n", rootHash[:16]+"...", len(chunks))
	for _, ch := range chunks {
		data, err := store.Get(ch)
		if err != nil {
			fmt.Printf("❌ Failed to fetch chunk %s: %v\n", ch, err)
			os.Exit(1)
		}
		outFile.Write(data)
	}
	fmt.Printf("✅ Fully restored to: %s\n", outPath)
}

func showStats(database *db.MetadataDB) {
	stats, err := database.GetStats()
	if err != nil {
		fmt.Printf("❌ Stats error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== 📈 CDC Dedup Engine Global Stats ===")
	fmt.Printf("📁 Total Files Stored : %v\n", stats["total_files"])
	fmt.Printf("🧱 Total Chunk Refs   : %v\n", stats["total_chunks"])
	fmt.Printf("✨ Unique Chunks      : %v\n", stats["unique_chunks"])
	fmt.Printf("💾 Logical File Size  : %s\n", humanize.Bytes(uint64(stats["logical_size"].(int64))))
	fmt.Printf("💽 Physical Disk Size : %s\n", humanize.Bytes(uint64(stats["physical_size"].(int64))))
	fmt.Printf("🏆 Global Dedup Ratio : %v%%\n", stats["dedup_ratio_percent"])
}

func showFileStats(filePath string, database *db.MetadataDB) {
	fileName := filepath.Base(filePath)
	stats, err := database.GetFileStatsByName(fileName)
	if err != nil {
		fmt.Printf("❌ Stats error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("=== 📈 File Stats for: %s ===\n", filePath)
	fmt.Printf("📄 File Name          : %v\n", stats["file_name"])
	fmt.Printf("🔑 Root Hash          : %v\n", stats["root_hash"])
	fmt.Printf("💾 File Size           : %s\n", humanize.Bytes(uint64(stats["file_size"].(int64))))
	fmt.Printf("🧱 Total Chunks        : %v\n", stats["total_chunks"])
	fmt.Printf("✨ Unique Chunks       : %v\n", stats["unique_chunks"])
	fmt.Printf("🏆 Global Dedup Ratio  : %v%%\n", stats["global_dedup_ratio"])
}

func runGC(store storage.Backend, database *db.MetadataDB) {
	fmt.Println("🧹 Running garbage collection...")
	count, err := database.RunGC(func(hash string) error {
		return store.Delete(hash)
	})
	if err != nil {
		fmt.Printf("❌ GC error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✨ GC completed! Cleaned up %d orphaned chunks.\n", count)
}
