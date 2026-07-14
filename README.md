# p2p-chat-cli

Encrypted P2P terminal chat client. Uses WebRTC DataChannels for direct peer-to-peer communication with end-to-end encryption. No message history, no server-side message storage — messages exist only in memory for currently connected peers.

## Architecture

### Encryption Layers

1. **X25519** — key exchange for room key distribution
2. **Ed25519** — message signing for sender identity verification
3. **AES-256-GCM** — symmetric message encryption with per-message random nonces
4. **Nonce replay protection** — each peer tracks seen nonces per session

### Flow

1. User registers with the backend, sending only public X25519/Ed25519 keys
2. Room creator generates a random AES-256 room key
3. Room key is encrypted to each member's X25519 public key (sealed box)
4. Only ciphertext is stored on the server
5. When entering a room, peers connect via WebRTC DataChannels
6. Every message is AES-256-GCM encrypted with a fresh nonce, then signed with Ed25519
7. Recipients verify the signature, check nonce uniqueness, then decrypt

## CLI Commands

```bash
chat register                           # Create a new account
chat login                              # Log in with existing credentials
chat logout                             # Clear local session
chat whoami                             # Show current user info

chat room create "Room Name"            # Create a new chat room
chat room list                          # List your rooms
chat room join INVITE_CODE              # Join via invite code
chat room leave "Room Name"             # Leave a room
chat room delete "Room Name"            # Delete a room (owner only)

chat invite "Room Name"                 # Generate an invite code

chat enter "Room Name"                  # Enter interactive chat session
```

### Interactive Session Commands

```
/members         — List connected peers
/help            — Show available commands
/exit            — Leave the room
```

## Setup

### Prerequisites

- Go 1.22+

### Build & Install

```bash
git clone <repo-url> p2p-chat-cli
cd p2p-chat-cli
make build
```

The binary will be placed at `./chat`. You can move it to your `$PATH`:

```bash
mv chat ~/.local/bin/  # or /usr/local/bin/
```

### Configuration

Created automatically at `~/.chat/config.json`:

```json
{
  "server_url": "http://localhost:8080",
  "jwt": "",
  "user_id": "",
  "username": "",
  "x25519_public_key": "...",
  "x25519_private_key": "...",
  "ed25519_public_key": "...",
  "ed25519_private_key": "..."
}
```

Set the server URL via environment variable:

```bash
export SERVER_URL=https://your-server.com
```

Or set it directly in the config file.

## Key Management

- X25519 and Ed25519 keypairs are generated locally on registration
- Private keys are stored in `~/.chat/config.json` with `0600` permissions
- Private keys **never** leave your machine
- Room AES keys are stored encrypted (sealed to your X25519 key) on the server
- On room entry, your local client decrypts the room key using your X25519 private key

## Makefile Targets

```bash
make build       # Build the CLI binary
make run         # Run the CLI (alias for build + execute)
make clean       # Remove build artifacts
make install     # Build and copy to $GOPATH/bin
```
