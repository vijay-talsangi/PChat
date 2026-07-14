package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var inviteCmd = &cobra.Command{
	Use:   "invite [room-name]",
	Short: "Generate an invite code for a room",
	Long:  "Generate an invite code (owner only). Default: 1 use, 48 hour expiry.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roomName := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		maxUses, _ := cmd.Flags().GetInt("max-uses")
		expiresHours, _ := cmd.Flags().GetInt("expires-hours")
		if maxUses < 1 {
			maxUses = 1
		}
		if expiresHours < 1 {
			expiresHours = 48
		}
		apiClient := newAPIClient(cfg.JWT)
		invite, err := apiClient.CreateInvite(roomName, maxUses, expiresHours)
		if err != nil {
			return fmt.Errorf("failed to create invite: %w", err)
		}
		fmt.Printf("Invite code: %s\n", invite.Code)
		fmt.Printf("  Room:      %s\n", roomName)
		fmt.Printf("  Max uses:  %d\n", invite.MaxUses)
		fmt.Printf("  Expires:   %s\n", invite.ExpiresAt)
		return nil
	},
}

func init() {
	inviteCmd.Flags().Int("max-uses", 1, "Maximum number of uses")
	inviteCmd.Flags().Int("expires-hours", 48, "Invite expiry in hours")
}
