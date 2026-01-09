package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type InfoResponse struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type FilesResponse struct {
	Files []FileMetadata `json:"files"`
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := InfoResponse{
		IP:   localIP,
		Port: port,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(100 << 20) // 100 MB max

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	expirationHours := 24 // Default to 24 hours
	if expStr := r.FormValue("expirationHours"); expStr != "" {
		if exp, err := json.Number(expStr).Int64(); err == nil && exp > 0 {
			expirationHours = int(exp)
		}
	}

	meta, err := storage.SaveFile(header.Filename, file, expirationHours)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files := storage.ListFiles()

	resp := FilesResponse{Files: files}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/download/")
	if id == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	meta, path, err := storage.GetFile(id)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+meta.Name+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/delete/")
	if id == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	if err := storage.DeleteFile(id); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
