package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vijay-talsangi/PChat/config"
)

var (
	cfg       *config.ConfigData
	version   = "dev"
	commit    = "unknown"
	date      = "unknown"
	debugMode bool
)

var rootCmd = &cobra.Command{
	Use:     "pchat",
	Short:   "P2P encrypted terminal chat",
	Version: version,
	Long: `End-to-end encrypted peer-to-peer chat over WebRTC DataChannels.
Messages are encrypted with AES-256-GCM and never stored server-side.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if os.Getenv("PCHAT_DEBUG") != "" {
			debugMode = true
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Show debug logs")
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(roomCmd)
	rootCmd.AddCommand(inviteCmd)
	rootCmd.AddCommand(enterCmd)
}
