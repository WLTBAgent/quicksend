package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/wltbagent/quicksend/storage"
)

type Handler struct {
	store *storage.Storage
}

func New(store *storage.Storage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-Key")
	if len(key) != 32 {
		http.Error(w, "invalid or missing X-Key header (must be 32 characters)", http.StatusUnauthorized)
		return
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/upload":
		h.upload(w, r, key)
	case r.Method == http.MethodGet && r.URL.Path == "/list":
		h.list(w, r, key)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/download/"):
		h.download(w, r, key)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request, key string) {
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := h.store.Upload(key, header.Filename, file); err != nil {
		log.Printf("upload error: %v", err)
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "uploaded")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, key string) {
	files := h.store.List(key)
	if files == nil {
		files = []storage.FileMeta{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request, key string) {
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	if filename == "" {
		http.Error(w, "missing file name", http.StatusBadRequest)
		return
	}

	result, err := h.store.Download(key, filename)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer result.File.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", result.Size))
	if _, err := io.Copy(w, result.File); err != nil {
		log.Printf("download write error: %v", err)
	}

	if err := h.store.Remove(key, filename); err != nil {
		log.Printf("failed to remove downloaded file %s/%s: %v", key, filename, err)
	}
}
