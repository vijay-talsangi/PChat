// Package config manages the local configuration file stored at ~/.pchat/config.json.
// It handles loading, saving, and providing defaults for the CLI application's
// persistent state including authentication tokens, cryptographic keys, and room keys.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// configDirName is the hidden directory name under the user's home directory.
	configDirName = ".pchat"
	// legacyConfigDirName preserves compatibility with older installs.
	legacyConfigDirName = ".chat"
	// configFileName is the name of the configuration file.
	configFileName = "config.json"
	// defaultServerURL is used when no SERVER_URL env var is set.
	defaultServerURL = "https://pchat-backend.onrender.com"
)

// ConfigData holds all persistent state for the CLI client.
type ConfigData struct {
	// ServerURL is the base URL of the backend server.
	ServerURL string `json:"server_url"`
	// JWT is the authentication token received after login.
	JWT string `json:"jwt,omitempty"`
	// UserID is the unique identifier for the authenticated user.
	UserID string `json:"user_id,omitempty"`
	// Username is the display name of the authenticated user.
	Username string `json:"username,omitempty"`
	// X25519PublicKey is the base64-encoded X25519 public key for key exchange.
	X25519PublicKey string `json:"x25519_public_key,omitempty"`
	// X25519PrivateKey is the base64-encoded X25519 private key for key exchange.
	X25519PrivateKey string `json:"x25519_private_key,omitempty"`
	// Ed25519PublicKey is the base64-encoded Ed25519 public key for message signing.
	Ed25519PublicKey string `json:"ed25519_public_key,omitempty"`
	// Ed25519PrivateKey is the base64-encoded Ed25519 private key for message signing.
	Ed25519PrivateKey string `json:"ed25519_private_key,omitempty"`
	// RoomKeys maps room names to their base64-encoded AES-256 symmetric keys.
	RoomKeys map[string]string `json:"room_keys,omitempty"`
}

// GetConfigPath returns the full path to the config file (~/.pchat/config.json).
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir cannot be determined.
		home = "."
	}
	return filepath.Join(home, configDirName, configFileName)
}

// getConfigDir returns the path to the config directory (~/.pchat/).
func getConfigDir() string {
	return filepath.Dir(GetConfigPath())
}

// getLegacyConfigPath returns the old config path used by previous releases.
func getLegacyConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, legacyConfigDirName, configFileName)
}

// EnsureConfigDir creates the config directory if it does not exist.
// The directory is created with 0700 permissions (owner read/write/execute only).
func EnsureConfigDir() error {
	dir := getConfigDir()
	return os.MkdirAll(dir, 0700)
}

// Load reads the configuration from ~/.pchat/config.json.
// If the file does not exist, a new ConfigData with default values is returned.
// The config directory is created if it doesn't already exist.
func Load() (*ConfigData, error) {
	if err := EnsureConfigDir(); err != nil {
		return nil, err
	}

	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			legacyPath := getLegacyConfigPath()
			data, err = os.ReadFile(legacyPath)
			if err != nil {
				if os.IsNotExist(err) {
					// Return a fresh config with defaults when no file exists.
					return defaultConfig(), nil
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	cfg := &ConfigData{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Always prefer the configured default server URL source over any saved value.
	cfg.ServerURL = resolveServerURL()

	// Ensure the room keys map is initialized.
	if cfg.RoomKeys == nil {
		cfg.RoomKeys = make(map[string]string)
	}

	return cfg, nil
}

// Save writes the configuration to ~/.pchat/config.json with 0600 permissions
// (owner read/write only) to protect sensitive data like private keys and tokens.
func Save(cfg *ConfigData) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigPath(), data, 0600)
}

// defaultConfig creates a new ConfigData populated with default values.
func defaultConfig() *ConfigData {
	return &ConfigData{
		ServerURL: resolveServerURL(),
		RoomKeys:  make(map[string]string),
	}
}

// resolveServerURL returns the server URL from the SERVER_URL environment
// variable, falling back to the default URL if the variable is not set.
func resolveServerURL() string {
	if url := os.Getenv("SERVER_URL"); url != "" {
		return url
	}
	return defaultServerURL
}
