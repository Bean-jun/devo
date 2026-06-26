package rest

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"devo/internal/core/session"
)

type fileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type getFilesResponse struct {
	Type    string      `json:"type"`
	Entries []fileEntry `json:"entries,omitempty"`
	Content string      `json:"content,omitempty"`
	Size    int64       `json:"size,omitempty"`
	IsImage bool        `json:"is_image,omitempty"`
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".svg": true, ".webp": true, ".bmp": true, ".ico": true,
}

func isImageFile(name string) bool {
	return imageExts[strings.ToLower(filepath.Ext(name))]
}

func (h *Handler) GetFiles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "."
	}

	absWorkDir, err := filepath.Abs(sess.WorkingDirectory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var absPath string
	if filepath.IsAbs(requestedPath) {
		absPath = requestedPath
	} else {
		absPath = filepath.Join(absWorkDir, requestedPath)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	if !strings.HasPrefix(absPath, absWorkDir) {
		writeError(w, http.StatusBadRequest, "path is outside working directory")
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path does not exist or is not accessible")
		return
	}

	if info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read directory")
			return
		}

		var fileEntries []fileEntry
		for _, entry := range entries {
			entryInfo, err := entry.Info()
			if err != nil {
				continue
			}
			entryType := "file"
			if entry.IsDir() {
				entryType = "dir"
			}
			fileEntries = append(fileEntries, fileEntry{
				Name: entry.Name(),
				Type: entryType,
				Size: entryInfo.Size(),
			})
		}

		writeJSON(w, http.StatusOK, getFilesResponse{
			Type:    "dir",
			Entries: fileEntries,
		})
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	img := isImageFile(info.Name())
	content := string(data)
	if img {
		content = "data:image/" + strings.TrimPrefix(filepath.Ext(info.Name()), ".") + ";base64," + base64.StdEncoding.EncodeToString(data)
	}

	writeJSON(w, http.StatusOK, getFilesResponse{
		Type:    "file",
		Content: content,
		Size:    info.Size(),
		IsImage: img,
	})
}
