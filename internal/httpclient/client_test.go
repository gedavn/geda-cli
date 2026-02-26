package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetPreservesQueryString(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	response, err := client.Get("/api/v1/posts?per_page=5&search=cli-test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response["ok"] != true {
		t.Fatalf("expected response ok=true, got %#v", response["ok"])
	}
	if capturedPath != "/api/v1/posts" {
		t.Fatalf("expected path /api/v1/posts, got %s", capturedPath)
	}
	if capturedQuery != "per_page=5&search=cli-test" {
		t.Fatalf("expected query per_page=5&search=cli-test, got %s", capturedQuery)
	}
}

func TestPostMultipartFileSendsFileAndFields(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(filePath, []byte("test-image"), 0o600); err != nil {
		t.Fatalf("failed to write temp image file: %v", err)
	}

	var uploadedName string
	var altVI string
	var altEN string
	var uploadedBytes int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to read multipart file: %v", err)
		}
		defer file.Close()

		fileBody, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}

		uploadedName = fileHeader.Filename
		uploadedBytes = len(fileBody)
		altVI = r.FormValue("alt_text[vi]")
		altEN = r.FormValue("alt_text[en]")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "ok",
		})
	}))
	defer server.Close()

	client := New(server.URL, "token")
	response, err := client.PostMultipartFile("/api/v1/media", "file", filePath, map[string]string{
		"alt_text[vi]": "Anh",
		"alt_text[en]": "Image",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response["message"] != "ok" {
		t.Fatalf("expected message ok, got %#v", response["message"])
	}
	if uploadedName != "image.png" {
		t.Fatalf("expected file name image.png, got %s", uploadedName)
	}
	if uploadedBytes != len([]byte("test-image")) {
		t.Fatalf("unexpected uploaded byte size %d", uploadedBytes)
	}
	if altVI != "Anh" || altEN != "Image" {
		t.Fatalf("unexpected alt text fields: vi=%q en=%q", altVI, altEN)
	}
}
