package commands

import (
	"encoding/json"
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
