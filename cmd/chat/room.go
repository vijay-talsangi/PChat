package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vijay-talsangi/PChat/api"
	"github.com/vijay-talsangi/PChat/config"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
)

var roomCmd = &cobra.Command{
	Use:   "room",
	Short: "Manage chat rooms",
}

var roomCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new chat room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		roomKey, err := pcrypto.GenerateRoomKey()
		if err != nil {
			return fmt.Errorf("failed to generate room key: %w", err)
		}
		cfg.RoomKeys[name] = pcrypto.EncodeBase64(roomKey)
		config.Save(cfg)
		selfEncodedKey, err := sealRoomKeyToSelf(roomKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt room key for self: %w", err)
		}
		apiClient := newAPIClient(cfg.JWT)
		room, err := apiClient.CreateRoom(name, selfEncodedKey)
		if err != nil {
			return fmt.Errorf("failed to create room: %w", err)
		}
		fmt.Printf("Room created: %s (id: %s)\n", room.Name, room.ID)
		return nil
	},
}

var roomListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your rooms",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		apiClient := newAPIClient(cfg.JWT)
		rooms, err := apiClient.ListRooms()
		if err != nil {
			return fmt.Errorf("failed to list rooms: %w", err)
		}
		if len(rooms) == 0 {
			fmt.Println("No rooms. Create one with 'chat room create <name>'")
			return nil
		}
		for _, r := range rooms {
			owner := ""
			if r.OwnerID == cfg.UserID {
				owner = " (owner)"
			}
			fmt.Printf("  %s  [%d members]%s\n", r.Name, r.MemberCount, owner)
		}
		return nil
	},
}

var roomJoinCmd = &cobra.Command{
	Use:   "join [invite-code]",
	Short: "Join a room using an invite code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		apiClient := newAPIClient(cfg.JWT)
		resp, err := apiClient.JoinRoom(code, "")
		if err != nil {
			return fmt.Errorf("failed to join room: %w", err)
		}
		fmt.Printf("Joined room (id: %s)\n", resp.RoomID)
		return nil
	},
}

var roomLeaveCmd = &cobra.Command{
	Use:   "leave [name]",
	Short: "Leave a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		apiClient := newAPIClient(cfg.JWT)
		if err := apiClient.LeaveRoom(name); err != nil {
			return fmt.Errorf("failed to leave room: %w", err)
		}
		delete(cfg.RoomKeys, name)
		config.Save(cfg)
		fmt.Printf("Left room '%s'\n", name)
		return nil
	},
}

var roomDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a room (owner only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		apiClient := newAPIClient(cfg.JWT)
		if err := apiClient.DeleteRoom(name); err != nil {
			return fmt.Errorf("failed to delete room: %w", err)
		}
		delete(cfg.RoomKeys, name)
		config.Save(cfg)
		fmt.Printf("Deleted room '%s'\n", name)
		return nil
	},
}

func init() {
	roomCmd.AddCommand(roomCreateCmd)
	roomCmd.AddCommand(roomListCmd)
	roomCmd.AddCommand(roomJoinCmd)
	roomCmd.AddCommand(roomLeaveCmd)
	roomCmd.AddCommand(roomDeleteCmd)
}

func sealRoomKeyToSelf(roomKey []byte) (string, error) {
	selfPub, err := pcrypto.DecodeBase64(cfg.X25519PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode own public key: %w", err)
	}
	selfPriv, err := pcrypto.DecodeBase64(cfg.X25519PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode own private key: %w", err)
	}
	sealed, err := pcrypto.SealRoomKey(roomKey, selfPub, selfPriv)
	if err != nil {
		return "", fmt.Errorf("failed to seal room key: %w", err)
	}
	return pcrypto.EncodeBase64(sealed), nil
}

func newAPIClient(token string) *api.Client {
	return api.NewClient(cfg.ServerURL, token)
}
