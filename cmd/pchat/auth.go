package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vijay-talsangi/PChat/config"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account",
	Long:  "Register a new user with the server. Generates X25519 and Ed25519 keypairs locally.",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		if username == "" || password == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
			fmt.Print("Password: ")
			fmt.Scanln(&password)
		}
		if username == "" || password == "" {
			return fmt.Errorf("username and password are required")
		}
		x25519Pub, x25519Priv, err := pcrypto.GenerateX25519Keypair()
		if err != nil {
			return fmt.Errorf("failed to generate X25519 keypair: %w", err)
		}
		ed25519Pub, ed25519Priv, err := pcrypto.GenerateEd25519Keypair()
		if err != nil {
			return fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
		}
		apiClient := newAPIClient("")
		resp, err := apiClient.Register(username, password,
			pcrypto.EncodeBase64(x25519Pub),
			pcrypto.EncodeBase64(ed25519Pub))
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
		cfg.JWT = resp.Token
		cfg.UserID = resp.User.ID
		cfg.Username = resp.User.Username
		cfg.X25519PublicKey = pcrypto.EncodeBase64(x25519Pub)
		cfg.X25519PrivateKey = pcrypto.EncodeBase64(x25519Priv)
		cfg.Ed25519PublicKey = pcrypto.EncodeBase64(ed25519Pub)
		cfg.Ed25519PrivateKey = pcrypto.EncodeBase64(ed25519Priv)
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Registered as %s\n", resp.User.Username)
		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to an existing account",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		if username == "" || password == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
			fmt.Print("Password: ")
			fmt.Scanln(&password)
		}
		if username == "" || password == "" {
			return fmt.Errorf("username and password are required")
		}
		apiClient := newAPIClient("")
		resp, err := apiClient.Login(username, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		cfg.JWT = resp.Token
		cfg.UserID = resp.User.ID
		cfg.Username = resp.User.Username
		// This is decision to allow only one trusted device
		// Login to new device with registered account in different device: NOT ALLOWED
		// Lost config and trying to login: NOT ALLOWED
		if cfg.X25519PublicKey == "" ||
			cfg.X25519PrivateKey == "" ||
			cfg.Ed25519PublicKey == "" ||
			cfg.Ed25519PrivateKey == "" {
			return fmt.Errorf("cryptographic identity is missing. This client supports only one trusted device. Please register a new account or restore your key backup")
		}
		// Trying to login with tampered keys: NOT ALLOWED
		if cfg.X25519PublicKey != resp.User.PublicKey {
			return fmt.Errorf("local identity does not match server identity")
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Logged in as %s\n", resp.User.Username)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local session data",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.JWT = ""
		cfg.UserID = ""
		cfg.Username = ""
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user info",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Username == "" {
			fmt.Println("Not logged in")
			return nil
		}
		fmt.Printf("Username: %s\n", cfg.Username)
		fmt.Printf("User ID:  %s\n", cfg.UserID)
		fmt.Printf("Server:   %s\n", cfg.ServerURL)
		return nil
	},
}

func init() {
	registerCmd.Flags().String("username", "", "Username")
	registerCmd.Flags().String("password", "", "Password")
	loginCmd.Flags().String("username", "", "Username")
	loginCmd.Flags().String("password", "", "Password")
}
