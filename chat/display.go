package chat

import (
	"fmt"
	"strings"
)

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

const headerWidth = 44

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func PrintHeader(roomName, username string) {
	inner := headerWidth - 4
	fmt.Println()
	fmt.Printf("╔%s╗\n", strings.Repeat("═", headerWidth-2))
	fmt.Printf("║ %s║\n", pad("  🔒 PChat", inner))
	fmt.Printf("║ %s║\n", pad(fmt.Sprintf("  Room: %s", roomName), inner))
	fmt.Printf("║ %s║\n", pad(fmt.Sprintf("  User: %s", username), inner))
	fmt.Printf("║ %s║\n", pad("  Status: 🟢 Connected", inner))
	fmt.Printf("╚%s╝\n", strings.Repeat("═", headerWidth-2))
	fmt.Println()
}

func PrintMessage(username, msg string) {
	fmt.Printf("%s[%s]%s %s\n", Bold, username, Reset, msg)
}

func PrintOwnMessage(msg string) {
	fmt.Printf("%sYou >%s %s\n", Bold, Reset, msg)
}

func PrintSystem(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func PrintWarning(msg string) {
	fmt.Printf("⚠ %s\n", msg)
}

func PrintConnected() {
	fmt.Println("🟢 Connected")
}

func PrintConnecting() {
	fmt.Println("🟡 Connecting...")
}

func PrintDisconnected() {
	fmt.Println("🔴 Disconnected")
}

func PrintReconnecting() {
	fmt.Println("⟳ Reconnecting...")
}

func PrintError(msg string) {
	fmt.Printf("❌ %s\n", msg)
}

func PrintHelp() {
	fmt.Println()
	fmt.Printf("%sCommands%s\n", Bold, Reset)
	fmt.Printf("  %-10s %s\n", "/help", "Show help")
	fmt.Printf("  %-10s %s\n", "/members", "Show online users")
	fmt.Printf("  %-10s %s\n", "/invite", "Invite a user")
	fmt.Printf("  %-10s %s\n", "/clear", "Clear chat")
	fmt.Printf("  %-10s %s\n", "/exit", "Leave room")
	fmt.Println()
}

func PrintMembers(members []string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════╗")
	fmt.Printf("║ %s%-20s%s║\n", Bold, "Online Users", Reset)
	fmt.Println("╠══════════════════════════╣")
	if len(members) == 0 {
		fmt.Printf("║  %-20s║\n", "No other users")
	} else {
		for _, m := range members {
			n := m
			if len(n) > 20 {
				n = n[:20]
			}
			fmt.Printf("║  🟢 %-17s║\n", n)
		}
	}
	fmt.Println("╚══════════════════════════╝")
	fmt.Println()
}

func ClearLine() {
	fmt.Print("\r\033[K")
}
