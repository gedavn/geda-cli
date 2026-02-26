package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndClearProfile(t *testing.T) {
	homeDir := t.TempDir()
	previousHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", previousHome)
	})

	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	profile := Profile{
		BaseURL:     "http://localhost:8000",
		AccessToken: "token-value",
		UserEmail:   "admin@example.com",
	}

	if err := Save(profile); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load profile: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected profile to be loaded")
	}

	if loaded.BaseURL != profile.BaseURL {
		t.Fatalf("expected base url %s, got %s", profile.BaseURL, loaded.BaseURL)
	}
	if loaded.AccessToken != profile.AccessToken {
		t.Fatalf("expected token %s, got %s", profile.AccessToken, loaded.AccessToken)
	}

	configPath := pathFromHome(homeDir)
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected config permissions 0600, got %o", fileInfo.Mode().Perm())
	}

	if err := Clear(); err != nil {
		t.Fatalf("failed to clear profile: %v", err)
	}

	if _, err := os.Stat(filepath.Clean(configPath)); !os.IsNotExist(err) {
		t.Fatalf("expected profile file to be removed, got: %v", err)
	}
}
