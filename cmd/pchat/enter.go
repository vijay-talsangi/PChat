package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vijay-talsangi/PChat/api"

	"github.com/vijay-talsangi/PChat/chat"
	"github.com/vijay-talsangi/PChat/config"
	pcrypto "github.com/vijay-talsangi/PChat/crypto"
)

var enterCmd = &cobra.Command{
	Use:   "enter [room-name]",
	Short: "Enter an interactive chat session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roomName := args[0]
		if cfg.JWT == "" {
			return fmt.Errorf("not logged in")
		}
		if cfg.UserID == "" {
			return fmt.Errorf("user ID not found in config")
		}
		roomKeyEncoded, ok := cfg.RoomKeys[roomName]
		if !ok {
			apiClient := newAPIClient(cfg.JWT)
			var encKey string
			var apiErr error
			fmt.Print("Waiting for room key from existing member...")
			for {
				encKey, apiErr = apiClient.GetRoomKey(roomName)
				if apiErr == nil {
					break
				}
				var statusErr *api.HTTPStatusError
				if !errors.As(apiErr, &statusErr) || statusErr.StatusCode != 404 {
					return fmt.Errorf("failed to fetch room key: %w", apiErr)
				}
				time.Sleep(500 * time.Millisecond)
			}
			fmt.Println(" OK")
			privKey, err := pcrypto.DecodeBase64(cfg.X25519PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to decode private key: %w", err)
			}
			sealed, err := pcrypto.DecodeBase64(encKey)
			if err != nil {
				return fmt.Errorf("failed to decode encrypted room key: %w", err)
			}
			roomKeyBytes, err := pcrypto.OpenRoomKey(sealed, privKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt room key: %w", err)
			}
			roomKeyEncoded = pcrypto.EncodeBase64(roomKeyBytes)
			cfg.RoomKeys[roomName] = roomKeyEncoded
			config.Save(cfg)
		}
		roomKey, err := pcrypto.DecodeBase64(roomKeyEncoded)
		if err != nil {
			return fmt.Errorf("failed to decode room key: %w", err)
		}
		signingKey, err := pcrypto.DecodeBase64(cfg.Ed25519PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to decode signing key: %w", err)
		}

		chat.PrintBanner(roomName)
		chat.PrintHelp()

		apiClient := newAPIClient(cfg.JWT)

		session := chat.NewSession(chat.SessionConfig{
			RoomName:    roomName,
			UserID:      cfg.UserID,
			Username:    cfg.Username,
			ServerURL:   cfg.ServerURL,
			Token:       cfg.JWT,
			RoomKey:     roomKey,
			SigningKey:  signingKey,
			APIClient:   apiClient,
			MembersFunc: fetchMembers,
		})

		return session.Start()
	},
}

func fetchMembers(roomName, token string) ([]chat.Member, error) {
	apiClient := newAPIClient(token)
	members, err := apiClient.GetRoomMembers(roomName)
	if err != nil {
		return nil, err
	}
	result := make([]chat.Member, len(members))
	for i, m := range members {
		edPub, _ := pcrypto.DecodeBase64(m.SigningPublicKey)
		result[i] = chat.Member{
			UserID:           m.UserID,
			Username:         m.Username,
			PublicKey:        m.PublicKey,
			SigningPublicKey: edPub,
		}
	}
	return result, nil
}
