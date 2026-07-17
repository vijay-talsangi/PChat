<p align="center">
  <img src="assets/banner.png" alt="PChat Banner" width="100%">
</p>

# PChat — P2P Encrypted Terminal Chat

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/vijay-talsangi/PChat)](https://github.com/vijay-talsangi/PChat/releases/latest)

**PChat** is an end-to-end encrypted peer-to-peer terminal chat client. It uses WebRTC DataChannels for direct peer-to-peer communication with no server-side message storage — messages exist only in memory for currently connected peers.

---

## Installation

### Option 1: Download a prebuilt binary (recommended)

Download the latest archive for your platform from the [Releases page](https://github.com/vijay-talsangi/PChat/releases/latest).

| Platform | Archive |
|----------|---------|
| Linux x86_64 | `pchat_Linux_x86_64.tar.gz` |
| Linux ARM64 | `pchat_Linux_arm64.tar.gz` |
| macOS Intel | `pchat_Darwin_x86_64.tar.gz` |
| macOS Apple Silicon | `pchat_Darwin_arm64.tar.gz` |
| Windows x86_64 | `pchat_Windows_x86_64.zip` |

**Linux / macOS**
```bash
tar -xzf pchat_*.tar.gz
sudo mv pchat /usr/local/bin/
```

**Windows** — Extract the `.zip` and add the folder to your `PATH`.

### Option 2: `go install`

```bash
go install github.com/vijay-talsangi/PChat/cmd/pchat@latest
```

The binary is placed in `$GOPATH/bin` (or `$HOME/go/bin` by default). Ensure that directory is in your `PATH`.

### Option 3: Build from source

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/vijay-talsangi/PChat.git
cd PChat
make build
```

The binary is placed at `./bin/pchat`. Move it to your `PATH`:

```bash
mv bin/pchat ~/.local/bin/
```

---

## Quick Start

1. **Register an account**
   ```bash
   pchat register
   ```
   X25519 and Ed25519 keypairs are generated locally. Private keys never leave your machine.

2. **Create a room**
   ```bash
   pchat room create "my-room"
   ```

3. **Generate an invite code** and share it with a friend
   ```bash
   pchat invite "my-room"
   ```

4. **Your friend joins** using the invite code
   ```bash
   pchat room join INVITE_CODE
   ```

5. **Enter the chat room**
   ```bash
   pchat enter "my-room"
   ```

6. **Type a message** and press Enter. It is encrypted end-to-end and sent over a direct WebRTC DataChannel.

---

## CLI Commands

```
pchat register                          Create a new account
pchat login                             Log in with existing credentials
pchat logout                            Clear local session
pchat whoami                            Show current user info

pchat room create "Room Name"           Create a new chat room
pchat room list                         List your rooms
pchat room join INVITE_CODE             Join via invite code
pchat room leave "Room Name"            Leave a room
pchat room delete "Room Name"           Delete a room (owner only)

pchat invite "Room Name"                Generate an invite code

pchat enter "Room Name"                 Enter interactive chat session
```

### Interactive Session Commands

```
/members         List connected peers
/help            Show available commands
/exit            Leave the room
```

---

## Architecture

### Encryption layers

1. **X25519** — key exchange for room key distribution
2. **Ed25519** — message signing for sender identity verification
3. **AES-256-GCM** — symmetric message encryption with per-message random nonces
4. **Nonce replay protection** — each peer tracks seen nonces per session

### Flow

1. User registers with the backend, sending only public X25519/Ed25519 keys
2. Room creator generates a random AES-256 room key
3. Room key is encrypted for each member's X25519 public key (sealed box)
4. Only ciphertext is stored on the server
5. When entering a room, peers connect via WebRTC DataChannels
6. Every message is AES-256-GCM encrypted with a fresh nonce, then signed with Ed25519
7. Recipients verify the signature, check nonce uniqueness, then decrypt

---

## Configuration

Configuration is stored at `~/.pchat/config.json`:

```json
{
  "server_url": "https://pchat-backend.onrender.com",
  "jwt": "",
  "user_id": "",
  "username": "",
  "x25519_public_key": "...",
  "x25519_private_key": "...",
  "ed25519_public_key": "...",
  "ed25519_private_key": "..."
}
```

Set the server URL via the `SERVER_URL` environment variable:

```bash
export SERVER_URL=https://your-server.com
```

---

## Key management

- X25519 and Ed25519 keypairs are generated locally on registration
- Private keys are stored in `~/.pchat/config.json` with `0600` permissions
- Private keys **never** leave your machine
- Room AES keys are stored encrypted (sealed to your X25519 key) on the server
- On room entry, your local client decrypts the room key using your X25519 private key

---

## Development

### Makefile targets

```bash
make build       # Build for current platform
make build-all   # Cross-compile for all supported platforms
make run         # Run the CLI
make clean       # Remove build artifacts
make install     # go install into $GOPATH/bin
make test        # Run tests
make lint        # Run go vet
```

### Cross-compilation

```bash
make build-all
```

Output binaries are placed in the `bin/` directory:

```
bin/pchat-linux-amd64
bin/pchat-linux-arm64
bin/pchat-darwin-amd64
bin/pchat-darwin-arm64
bin/pchat-windows-amd64.exe
```

---

## License

[MIT](LICENSE)
