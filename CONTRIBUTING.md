# Contributing to PChat

Thank you for your interest in contributing! PChat is an end-to-end encrypted peer-to-peer terminal chat client. This document will help you get started.

## Table of Contents

- [Project Structure](#project-structure)
- [Fork and Clone](#fork-and-clone)
- [Building the CLI](#building-the-cli)
- [Running Tests](#running-tests)
- [End-to-End Testing on Two Systems](#end-to-end-testing-on-two-systems)
- [Submitting Changes](#submitting-changes)

## Project Structure

```
PChat/
├── api/           # HTTP + WebSocket client for backend
├── chat/          # Interactive chat session
├── cmd/pchat/     # CLI entry point (cobra commands)
├── config/        # Local config (~/.pchat/config.json)
├── crypto/        # E2E encryption (AES, X25519, Ed25519)
├── rtc/           # WebRTC peer connection + messages
├── CONTRIBUTING.md
├── Makefile
├── README.md
└── ...
```

The CLI connects to a hosted signaling server at `https://pchat-backend.onrender.com` (configurable via `SERVER_URL` env var). You don't need to run the backend yourself.

## Fork and Clone

1. Fork the repository on GitHub.
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/PChat.git
   cd PChat
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/vijay-talsangi/PChat.git
   ```
4. Create a feature branch:
   ```bash
   git checkout -b feat/your-feature-name
   ```

## Building the CLI

### Prerequisites

- Go 1.22+ ([install](https://go.dev/dl/))

### Build

```bash
make build
```

The binary is placed at `./bin/pchat`. You can also install it:

```bash
make install
```

### Makefile targets

| Target | Description |
|--------|-------------|
| `make build` | Build for current platform |
| `make build-all` | Cross-compile for all supported platforms |
| `make run` | Build and run the CLI |
| `make clean` | Remove build artifacts |
| `make install` | `go install` into `$GOPATH/bin` |
| `make test` | Run all tests |
| `make lint` | Run `go vet ./...` |

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

### Quick start

```bash
./bin/pchat register --username alice --password secret123
./bin/pchat login --username alice --password secret123
./bin/pchat room create my-room
./bin/pchat invite my-room
./bin/pchat enter my-room
```

## Running Tests

Run all tests:

```bash
make test        # or: go test ./... -v
```

Run tests for a specific package:

```bash
go test ./crypto/... -v
go test ./config/... -v
```

Run tests with race detection:

```bash
go test -race ./...
```

## End-to-End Testing on Two Systems

To verify that the chat works across different machines you will need two systems (or two terminal sessions). The CLI connects to the hosted signaling server at `https://pchat-backend.onrender.com` by default — no backend setup required.

### System A (creates the room)

```bash
make build

./bin/pchat register --username alice --password pass123
./bin/pchat login --username alice --password pass123

./bin/pchat room create my-room
./bin/pchat invite my-room --max-uses 1 --expires-hours 24

# Note the invite code (e.g., SILVER-WOLF-42) and share it with System B

./bin/pchat enter my-room
```

### System B (joins the room)

```bash
make build

./bin/pchat register --username bob --password pass456
./bin/pchat login --username bob --password pass456

./bin/pchat room join SILVER-WOLF-42

./bin/pchat enter my-room
```

Once both users run `pchat enter`, they will connect via WebRTC. Type messages and press Enter to send. Use `/help` for available commands.

## Submitting Changes

1. Commit your changes with a descriptive message:
   ```bash
   git add .
   git commit -m "feat: add support for X feature"
   ```

2. Push to your fork:
   ```bash
   git push origin feat/your-feature-name
   ```

3. Open a Pull Request against the `main` branch of the upstream repository.

4. Ensure all CI checks pass and maintain test coverage for any new code.

### Commit Style

We follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new feature
- `fix:` bug fix
- `test:` adding or updating tests
- `docs:` documentation changes
- `refactor:` code restructuring
- `chore:` build/config changes

### Code Style

- Run `go vet ./...` before committing
- Follow existing code style and conventions
- Add tests for new functionality
- Keep functions focused and modular
- Use meaningful variable names
