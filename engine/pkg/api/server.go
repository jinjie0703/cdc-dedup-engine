package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jinjie0703/cdc-dedup-engine/pkg/db"
	"github.com/jinjie0703/cdc-dedup-engine/pkg/storage"
)

type Server struct {
	db      *db.MetadataDB
	store   storage.Backend
	port    int
}

func NewServer(database *db.MetadataDB, storeBackend storage.Backend, port int) *Server {
	return &Server{db: database, store: storeBackend, port: port}
}

func (s *Server) enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": stats,
	})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	files, err := s.db.ListFiles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": files,
	})
}

func (s *Server) handleGC(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	// 执行 GC，清理物理文件与元数据
	count, err := s.db.RunGC(func(hash string) error {
		return s.store.Delete(hash)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  fmt.Sprintf("GC completed, cleaned %d orphaned chunks", count),
	})
}

func (s *Server) Start() error {
	http.HandleFunc("/api/stats", s.handleStats)
	http.HandleFunc("/api/files", s.handleFiles)
	http.HandleFunc("/api/gc", s.handleGC)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🚀 API Server listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
