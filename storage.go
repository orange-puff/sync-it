package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FileMetadata struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploadedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type FileStorage struct {
	dir          string
	metadataFile string
	files        []FileMetadata
	mu           sync.RWMutex
}

func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create uploads directory: %w", err)
	}

	fs := &FileStorage{
		dir:          dir,
		metadataFile: filepath.Join(dir, "metadata.json"),
		files:        []FileMetadata{},
	}

	if err := fs.loadMetadata(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileStorage) loadMetadata() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.metadataFile)
	if os.IsNotExist(err) {
		fs.files = []FileMetadata{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if err := json.Unmarshal(data, &fs.files); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	return nil
}

func (fs *FileStorage) saveMetadata() error {
	data, err := json.MarshalIndent(fs.files, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(fs.metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (fs *FileStorage) SaveFile(filename string, r io.Reader, expirationHours int) (*FileMetadata, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	id := generateID()
	storedPath := filepath.Join(fs.dir, id)

	f, err := os.Create(storedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	size, err := io.Copy(f, r)
	if err != nil {
		os.Remove(storedPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expirationHours) * time.Hour)

	meta := FileMetadata{
		ID:         id,
		Name:       filename,
		Size:       size,
		UploadedAt: now,
		ExpiresAt:  expiresAt,
	}

	fs.files = append(fs.files, meta)

	if err := fs.saveMetadata(); err != nil {
		os.Remove(storedPath)
		fs.files = fs.files[:len(fs.files)-1]
		return nil, err
	}

	return &meta, nil
}

func (fs *FileStorage) ListFiles() []FileMetadata {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make([]FileMetadata, len(fs.files))
	copy(result, fs.files)

	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt.After(result[j].UploadedAt)
	})

	return result
}

func (fs *FileStorage) GetFile(id string) (*FileMetadata, string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for _, meta := range fs.files {
		if meta.ID == id {
			path := filepath.Join(fs.dir, id)
			if _, err := os.Stat(path); err != nil {
				return nil, "", fmt.Errorf("file not found on disk")
			}
			return &meta, path, nil
		}
	}

	return nil, "", fmt.Errorf("file not found")
}

func (fs *FileStorage) DeleteFile(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	idx := -1
	for i, meta := range fs.files {
		if meta.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("file not found")
	}

	path := filepath.Join(fs.dir, id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	fs.files = append(fs.files[:idx], fs.files[idx+1:]...)

	if err := fs.saveMetadata(); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) ClearAllFiles() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, meta := range fs.files {
		path := filepath.Join(fs.dir, meta.ID)
		os.Remove(path)
	}

	fs.files = []FileMetadata{}

	if err := fs.saveMetadata(); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) DeleteExpiredFiles() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	now := time.Now()
	var activeFiles []FileMetadata

	for _, meta := range fs.files {
		if now.After(meta.ExpiresAt) {
			// File has expired, delete it
			path := filepath.Join(fs.dir, meta.ID)
			os.Remove(path)
		} else {
			// File is still active
			activeFiles = append(activeFiles, meta)
		}
	}

	fs.files = activeFiles

	if err := fs.saveMetadata(); err != nil {
		return err
	}

	return nil
}
