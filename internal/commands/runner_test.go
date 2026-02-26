package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"geda-cli/internal/config"
)

func TestUnknownCommandReturnsValidationExitCode(t *testing.T) {
	exitCode := Run([]string{"unknown"})
	if exitCode != ExitValidation {
		t.Fatalf("expected exit code %d, got %d", ExitValidation, exitCode)
	}
}

func TestAuthLoginSavesProfile(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" || r.Method != http.MethodPost {
			http.NotFound(w, r)

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "Bearer",
			"user": map[string]any{
				"email": "admin@example.com",
			},
		})
	}))
	defer server.Close()

	exitCode := Run([]string{
		"auth", "login",
		"--base-url", server.URL,
		"--email", "admin@example.com",
		"--password", "password123",
	})

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	profilePath := filepath.Join(homeDir, ".config", "geda-cli", "config.json")
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("expected profile file to be created: %v", err)
	}

	profile, err := config.Load()
	if err != nil {
		t.Fatalf("expected profile to load: %v", err)
	}
	if profile == nil || profile.AccessToken != "test-token" {
		t.Fatalf("expected token test-token, got %+v", profile)
	}
}

func TestWhoamiReturnsAuthExitCodeOn401(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/me" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":    "Unauthenticated.",
				"error_code": "unauthenticated",
			})

			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL:     server.URL,
		AccessToken: "expired-token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"auth", "whoami"})
	if exitCode != ExitAuth {
		t.Fatalf("expected exit code %d, got %d", ExitAuth, exitCode)
	}
}

func TestWhoamiReturnsSuccess(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" || r.Method != http.MethodGet {
			http.NotFound(w, r)

			return
		}
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":    "Unauthenticated.",
				"error_code": "unauthenticated",
			})

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":    1,
				"email": "admin@example.com",
			},
		})
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL:     server.URL,
		AccessToken: "valid-token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"auth", "whoami"})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
}

func TestHealthCheckWithBaseURLReturnsSuccess(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" || r.Method != http.MethodGet {
			http.NotFound(w, r)

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "ok",
			"app":         "GEDA",
			"environment": "test",
		})
	}))
	defer server.Close()

	exitCode := Run([]string{"health", "check", "--base-url", server.URL})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
}

func TestHealthCheckUsesSavedProfileBaseURL(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" || r.Method != http.MethodGet {
			http.NotFound(w, r)

			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
		})
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL: server.URL,
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"health", "check"})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
}

func TestHealthCheckReturnsValidationWhenBaseURLMissingAndNoProfile(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	exitCode := Run([]string{"health", "check"})
	if exitCode != ExitValidation {
		t.Fatalf("expected exit code %d, got %d", ExitValidation, exitCode)
	}
}

func TestHealthCheckReturnsNetworkExitCodeWhenResponseIsNotJSON(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" || r.Method != http.MethodGet {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>fatal error</body></html>"))
	}))
	defer server.Close()

	exitCode := Run([]string{"health", "check", "--base-url", server.URL})
	if exitCode != ExitNetwork {
		t.Fatalf("expected exit code %d, got %d", ExitNetwork, exitCode)
	}
}

func TestPostUpsertCreatesWhenPostDoesNotExist(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	payloadFile := writePayloadFile(t, map[string]any{
		"slug": "post-create-test",
		"title": map[string]any{
			"vi": "Bai viet moi",
			"en": "New post",
		},
		"body": map[string]any{
			"vi": "Noi dung VI",
			"en": "EN content",
		},
		"category_id": 1,
		"status":      "draft",
	})

	var postedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/posts/post-create-test":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":    "Not found",
				"error_code": "not_found",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/posts":
			if err := json.NewDecoder(r.Body).Decode(&postedBody); err != nil {
				t.Fatalf("failed to decode posted body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "Post created successfully.",
				"data": map[string]any{
					"slug": "post-create-test",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL:     server.URL,
		AccessToken: "valid-token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"post", "upsert", "--file", payloadFile})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	if postedBody == nil {
		t.Fatal("expected POST /api/v1/posts to be called")
	}
	if postedBody["slug"] != "post-create-test" {
		t.Fatalf("expected slug post-create-test, got %#v", postedBody["slug"])
	}

	title, ok := postedBody["title"].(map[string]any)
	if !ok {
		t.Fatalf("expected title object, got %#v", postedBody["title"])
	}
	if title["vi"] != "Bai viet moi" || title["en"] != "New post" {
		t.Fatalf("unexpected title payload: %#v", title)
	}
}

func TestPostUpsertUpdatesWhenPostExists(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	payloadFile := writePayloadFile(t, map[string]any{
		"slug": "post-update-test",
		"title": map[string]any{
			"vi": "Bai viet cap nhat",
			"en": "Updated post",
		},
		"body": map[string]any{
			"vi": "Noi dung VI updated",
			"en": "EN content updated",
		},
		"category_id": 1,
		"status":      "published",
	})

	var putBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/posts/post-update-test":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"slug": "post-update-test",
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/posts/post-update-test":
			if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
				t.Fatalf("failed to decode put body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "Post updated successfully.",
				"data": map[string]any{
					"slug": "post-update-test",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL:     server.URL,
		AccessToken: "valid-token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"post", "upsert", "--file", payloadFile})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	if putBody == nil {
		t.Fatal("expected PUT /api/v1/posts/post-update-test to be called")
	}
	if putBody["status"] != "published" {
		t.Fatalf("expected status published, got %#v", putBody["status"])
	}
}

func TestPostUploadImageReturnsSuccess(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	imageFile := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(imageFile, []byte("fake-image-bytes"), 0o600); err != nil {
		t.Fatalf("failed to create image file: %v", err)
	}

	var receivedFileName string
	var receivedFileSize int
	var receivedAltVI string
	var receivedAltEN string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/media" || r.Method != http.MethodPost {
			http.NotFound(w, r)

			return
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to read file field: %v", err)
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}

		receivedFileName = fileHeader.Filename
		receivedFileSize = len(fileBytes)
		receivedAltVI = r.FormValue("alt_text[vi]")
		receivedAltEN = r.FormValue("alt_text[en]")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "Media uploaded successfully.",
			"data": map[string]any{
				"id":   10,
				"url":  "http://example.com/storage/media/2026/02/image.png",
				"path": "media/2026/02/image.png",
			},
		})
	}))
	defer server.Close()

	if err := config.Save(config.Profile{
		BaseURL:     server.URL,
		AccessToken: "valid-token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode := Run([]string{"post", "upload-image", "--file", imageFile, "--alt-vi", "Anh bai viet", "--alt-en", "Post image"})
	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}

	if receivedFileName != "image.png" {
		t.Fatalf("expected uploaded filename image.png, got %s", receivedFileName)
	}
	if receivedFileSize != len([]byte("fake-image-bytes")) {
		t.Fatalf("unexpected uploaded file size: %d", receivedFileSize)
	}
	if receivedAltVI != "Anh bai viet" || receivedAltEN != "Post image" {
		t.Fatalf("unexpected alt_text values: vi=%q en=%q", receivedAltVI, receivedAltEN)
	}
}

func TestPostUploadImageRequiresFile(t *testing.T) {
	homeDir := t.TempDir()
	setTempHome(t, homeDir)

	exitCode := Run([]string{"post", "upload-image"})
	if exitCode != ExitAuth {
		t.Fatalf("expected exit code %d, got %d", ExitAuth, exitCode)
	}

	if err := config.Save(config.Profile{
		BaseURL:     "http://example.test",
		AccessToken: "token",
		UserEmail:   "admin@example.com",
	}); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	exitCode = Run([]string{"post", "upload-image"})
	if exitCode != ExitValidation {
		t.Fatalf("expected exit code %d, got %d", ExitValidation, exitCode)
	}
}

func writePayloadFile(t *testing.T, payload map[string]any) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "payload.json")
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	if err := os.WriteFile(filePath, payloadBytes, 0o600); err != nil {
		t.Fatalf("failed to write payload file: %v", err)
	}

	return filePath
}

func setTempHome(t *testing.T, homeDir string) {
	t.Helper()

	previousHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", previousHome)
	})

	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
}
