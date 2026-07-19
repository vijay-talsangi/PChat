package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Fatal("defaultConfig returned nil")
	}
	if cfg.ServerURL != defaultServerURL {
		t.Errorf("expected default server URL %q, got %q", defaultServerURL, cfg.ServerURL)
	}
	if cfg.RoomKeys == nil {
		t.Fatal("expected non-nil RoomKeys map")
	}
	if len(cfg.RoomKeys) != 0 {
		t.Errorf("expected empty RoomKeys, got %d entries", len(cfg.RoomKeys))
	}
}

func TestConfigPathNotEmpty(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Fatal("GetConfigPath returned empty string")
	}
}

func TestServerURLFromEnv(t *testing.T) {
	original := os.Getenv("SERVER_URL")
	defer os.Setenv("SERVER_URL", original)

	os.Setenv("SERVER_URL", "http://custom:9090")
	url := resolveServerURL()
	if url != "http://custom:9090" {
		t.Errorf("expected http://custom:9090, got %s", url)
	}
}

func TestServerURLDefault(t *testing.T) {
	original := os.Getenv("SERVER_URL")
	defer os.Setenv("SERVER_URL", original)

	os.Unsetenv("SERVER_URL")
	url := resolveServerURL()
	if url != defaultServerURL {
		t.Errorf("expected default %q, got %q", defaultServerURL, url)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	cfg := &ConfigData{
		ServerURL:        "http://test:8080",
		JWT:              "test-jwt-token",
		UserID:           "user-123",
		Username:         "testuser",
		X25519PublicKey:  "x25519-pub",
		X25519PrivateKey: "x25519-priv",
		Ed25519PublicKey: "ed25519-pub",
		Ed25519PrivateKey: "ed25519-priv",
		RoomKeys: map[string]string{
			"test-room": "room-key-data",
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.JWT != "test-jwt-token" {
		t.Errorf("JWT mismatch: got %q", loaded.JWT)
	}
	if loaded.UserID != "user-123" {
		t.Errorf("UserID mismatch: got %q", loaded.UserID)
	}
	if loaded.Username != "testuser" {
		t.Errorf("Username mismatch: got %q", loaded.Username)
	}
	if loaded.RoomKeys["test-room"] != "room-key-data" {
		t.Errorf("RoomKeys mismatch: got %q", loaded.RoomKeys["test-room"])
	}
}

func TestLoadNonExistentReturnsDefault(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load for non-existent config failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil for non-existent config")
	}
	if cfg.ServerURL != resolveServerURL() {
		t.Errorf("expected server URL %q, got %q", resolveServerURL(), cfg.ServerURL)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	cfg := &ConfigData{ServerURL: "http://test:8080"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(tmpHome, configDirName, configFileName))
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected 0600 permissions, got %o", perm)
	}
}

func TestEnsureConfigDirCreatesDir(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	dir := getConfigDir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf("expected 0700 permissions, got %o", perm)
	}
}
