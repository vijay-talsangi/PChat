// Package chat — terminal display helpers with ANSI color support.
// Provides formatted output functions for system messages, user messages,
// errors, and decorative elements in the interactive chat session.
package chat

import (
	"fmt"
	"strings"
)

// ANSI color and style escape codes.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// PrintSystem prints a system notification in yellow with a [system] prefix.
func PrintSystem(msg string) {
	fmt.Printf("%s%s[system]%s %s\n", ColorBold, ColorYellow, ColorReset, msg)
}

// PrintMessage prints an incoming chat message with the sender's username
// in cyan and the message text in white.
func PrintMessage(username, msg string) {
	fmt.Printf("%s%s%s%s: %s\n", ColorBold, ColorCyan, username, ColorReset, msg)
}

// PrintOwnMessage prints the user's own message with a [you] prefix in green.
func PrintOwnMessage(msg string) {
	fmt.Printf("%s%s[you]%s %s\n", ColorBold, ColorGreen, ColorReset, msg)
}

// PrintError prints an error message in red with an [error] prefix.
func PrintError(msg string) {
	fmt.Printf("%s%s[error]%s %s\n", ColorBold, ColorRed, ColorReset, msg)
}

// PrintMembers displays a formatted list of connected peer usernames.
func PrintMembers(members []string) {
	fmt.Printf("\n%s%s╔══════════════════════════════════╗%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s║       Connected Members          ║%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s╠══════════════════════════════════╣%s\n", ColorBold, ColorCyan, ColorReset)
	if len(members) == 0 {
		fmt.Printf("%s%s║  No other members connected      ║%s\n", ColorDim, ColorCyan, ColorReset)
	} else {
		for _, m := range members {
			// Pad the member name to fit within the box.
			padded := fmt.Sprintf("  ● %-29s", m)
			if len(padded) > 34 {
				padded = padded[:34]
			}
			fmt.Printf("%s%s║%s%s%s%s║%s\n", ColorBold, ColorCyan, ColorReset, ColorGreen, padded, ColorCyan, ColorReset)
		}
	}
	fmt.Printf("%s%s╚══════════════════════════════════╝%s\n\n", ColorBold, ColorCyan, ColorReset)
}

// PrintHelp displays the available chat commands.
func PrintHelp() {
	fmt.Printf("\n%s%sAvailable Commands:%s\n", ColorBold, ColorYellow, ColorReset)
	fmt.Printf("  %s/members%s  — List connected peers\n", ColorCyan, ColorReset)
	fmt.Printf("  %s/help%s     — Show this help message\n", ColorCyan, ColorReset)
	fmt.Printf("  %s/exit%s     — Leave the room\n", ColorCyan, ColorReset)
	fmt.Printf("\n  Type anything else to send a message.\n\n")
}

// ClearLine clears the current terminal line using ANSI escape codes.
func ClearLine() {
	fmt.Print("\r\033[K")
}

// PrintBanner displays a decorative ASCII art banner when entering a chat room.
func PrintBanner(roomName string) {
	width := 50
	// Build the room name display line, centered within the banner.
	nameDisplay := fmt.Sprintf("Room: %s", roomName)
	padding := (width - 4 - len(nameDisplay)) / 2
	if padding < 0 {
		padding = 0
	}
	paddedName := fmt.Sprintf("%s%s%s",
		strings.Repeat(" ", padding),
		nameDisplay,
		strings.Repeat(" ", width-4-padding-len(nameDisplay)))
	if len(paddedName) > width-4 {
		paddedName = paddedName[:width-4]
	}

	fmt.Println()
	fmt.Printf("%s%s", ColorBold, ColorCyan)
	fmt.Printf("  ╔%s╗\n", strings.Repeat("═", width-4))
	fmt.Printf("  ║%s║\n", strings.Repeat(" ", width-4))
	fmt.Printf("  ║%s%s%s%s%s║\n", ColorYellow, paddedName, ColorCyan, ColorBold, "")
	fmt.Printf("  ║%s║\n", strings.Repeat(" ", width-4))
	fmt.Printf("  ║%s  🔒 End-to-End Encrypted P2P Chat%s║\n",
		ColorGreen,
		strings.Repeat(" ", width-4-35)+ColorCyan+ColorBold)
	fmt.Printf("  ║%s║\n", strings.Repeat(" ", width-4))
	fmt.Printf("  ╚%s╝\n", strings.Repeat("═", width-4))
	fmt.Printf("%s\n", ColorReset)
	fmt.Printf("  %sType /help for available commands.%s\n\n", ColorDim, ColorReset)
}
