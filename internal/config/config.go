package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Profile struct {
	BaseURL     string `json:"base_url"`
	AccessToken string `json:"access_token"`
	UserEmail   string `json:"user_email"`
	LastLoginAt string `json:"last_login_at"`
}

func pathFromHome(home string) string {
	return filepath.Join(home, ".config", "geda-cli", "config.json")
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return pathFromHome(home), nil
}

func Load() (*Profile, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

func Save(profile Profile) error {
	path, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	if profile.LastLoginAt == "" {
		profile.LastLoginAt = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
