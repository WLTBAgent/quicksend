package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type FileMeta struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Storage struct {
	mu       sync.Mutex
	baseDir  string
	metaFile string
	files    map[string][]FileMeta
}

func New(baseDir string) (*Storage, error) {
	s := &Storage{
		baseDir:  baseDir,
		metaFile: filepath.Join(baseDir, "meta.json"),
		files:    make(map[string][]FileMeta),
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := s.loadMeta(); err != nil {
		return nil, fmt.Errorf("load meta: %w", err)
	}
	return s, nil
}

func (s *Storage) Upload(key, filename string, data io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyDir := filepath.Join(s.baseDir, key)
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}

	safeName := safeFilename(filename)
	filePath := filepath.Join(keyDir, safeName)

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, data)
	if err != nil {
		os.Remove(filePath)
		return fmt.Errorf("write file: %w", err)
	}

	meta := FileMeta{
		Name:      safeName,
		Size:      n,
		UploadedAt: time.Now(),
	}

	s.files[key] = append(s.files[key], meta)
	return s.saveMeta()
}

func (s *Storage) List(key string) []FileMeta {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]FileMeta, len(s.files[key]))
	copy(result, s.files[key])
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})
	return result
}

type DownloadResult struct {
	File     *os.File
	Size     int64
	Filename string
}

func (s *Storage) Download(key, filename string) (*DownloadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyDir := filepath.Join(s.baseDir, key)
	filePath := filepath.Join(keyDir, filename)

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return &DownloadResult{
		File:     f,
		Size:     fi.Size(),
		Filename: filename,
	}, nil
}

func (s *Storage) Remove(key, filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.removeUnlocked(key, filename)
}

func (s *Storage) removeUnlocked(key, filename string) error {
	keyDir := filepath.Join(s.baseDir, key)
	filePath := filepath.Join(keyDir, filename)
	os.Remove(filePath)

	metas := s.files[key]
	for i, m := range metas {
		if m.Name == filename {
			s.files[key] = append(metas[:i], metas[i+1:]...)
			break
		}
	}
	if len(s.files[key]) == 0 {
		delete(s.files, key)
	}
	return s.saveMeta()
}

func (s *Storage) PurgeExpired(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for key, metas := range s.files {
		var remaining []FileMeta
		for _, m := range metas {
			if m.UploadedAt.Before(cutoff) {
				keyDir := filepath.Join(s.baseDir, key)
				os.Remove(filepath.Join(keyDir, m.Name))
				removed++
			} else {
				remaining = append(remaining, m)
			}
		}
		if len(remaining) == 0 {
			delete(s.files, key)
		} else {
			s.files[key] = remaining
		}
	}
	if removed > 0 {
		s.saveMeta()
	}
	return removed
}

func (s *Storage) loadMeta() error {
	data, err := os.ReadFile(s.metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.files)
}

func (s *Storage) saveMeta() error {
	data, err := json.Marshal(s.files)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaFile, data, 0644)
}

func safeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		name = "file"
	}
	return name
}
