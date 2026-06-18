package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"devo/internal/core/session"
)

func TestGetFiles_SuccessFile(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	expectedContent := "Hello, this is test content."
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(expectedContent), 0644)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-1/files?path=test.txt")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result getFilesResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Type != "file" {
		t.Errorf("expected type 'file', got %q", result.Type)
	}
	if result.Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, result.Content)
	}
	if result.Size != int64(len(expectedContent)) {
		t.Errorf("expected size %d, got %d", len(expectedContent), result.Size)
	}
}

func TestGetFiles_SuccessDirectory(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-1/files")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result getFilesResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Type != "dir" {
		t.Errorf("expected type 'dir', got %q", result.Type)
	}
	if len(result.Entries) != 3 {
		t.Errorf("expected 3 entries (a.txt, b.txt, subdir), got %d", len(result.Entries))
	}

	hasA := false
	hasB := false
	hasSubdir := false
	for _, entry := range result.Entries {
		if entry.Name == "a.txt" && entry.Type == "file" {
			hasA = true
		}
		if entry.Name == "b.txt" && entry.Type == "file" {
			hasB = true
		}
		if entry.Name == "subdir" && entry.Type == "dir" {
			hasSubdir = true
		}
	}

	if !hasA || !hasB || !hasSubdir {
		t.Error("missing expected entries in directory listing")
	}
}

func TestGetFiles_PathOutsideWorkDir(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-1/files?path=../../etc/passwd")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for path outside working directory, got %d", resp.StatusCode)
	}
}

func TestGetFiles_PathNotFound(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-1/files?path=nonexistent.txt")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for non-existent path, got %d", resp.StatusCode)
	}
}

func TestGetFiles_SessionNotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/sessions/nonexistent/files")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for non-existent session, got %d", resp.StatusCode)
	}
}

func TestGetFiles_Subdirectory(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir1/dir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "dir1/dir2/file.txt"), []byte("nested"), 0644)

	sess := &session.Session{
		ID:               "sess-1",
		Title:            "Test",
		WorkingDirectory: tmpDir,
		State:            session.StateIdle,
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-1/files?path=dir1/dir2")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result getFilesResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Type != "dir" {
		t.Errorf("expected dir type, got %q", result.Type)
	}
}
