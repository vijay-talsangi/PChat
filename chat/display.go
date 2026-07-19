package chat

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const headerWidth = 44

var userColorPalette = []lipgloss.Color{
	lipgloss.Color("#FF6B6B"),
	lipgloss.Color("#51CF66"),
	lipgloss.Color("#FFD43B"),
	lipgloss.Color("#4DABF7"),
	lipgloss.Color("#DA77F2"),
	lipgloss.Color("#FF922B"),
	lipgloss.Color("#F783AC"),
	lipgloss.Color("#748FFC"),
	lipgloss.Color("#69DB7C"),
	lipgloss.Color("#FCC419"),
	lipgloss.Color("#22B8CF"),
	lipgloss.Color("#FF8787"),
}

func hashColor(s string) lipgloss.Color {
	h := fnv.New32a()
	h.Write([]byte(s))
	return userColorPalette[h.Sum32()%uint32(len(userColorPalette))]
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func RenderHeader(roomName, username, status string) string {
	inner := headerWidth - 4
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("╔%s╗\n", strings.Repeat("═", headerWidth-2)))
	b.WriteString(fmt.Sprintf("║ %s║\n", pad("  🔒 PChat", inner)))
	b.WriteString(fmt.Sprintf("║ %s║\n", pad(fmt.Sprintf("  Room: %s", roomName), inner)))
	b.WriteString(fmt.Sprintf("║ %s║\n", pad(fmt.Sprintf("  User: %s", username), inner)))
	b.WriteString(fmt.Sprintf("║ %s║\n", pad(fmt.Sprintf("  Status: %s", status), inner)))
	b.WriteString(fmt.Sprintf("╚%s╝\n", strings.Repeat("═", headerWidth-2)))
	return b.String()
}

func StylePeerMessage(username, text string) string {
	color := hashColor(username)
	name := lipgloss.NewStyle().Bold(true).Foreground(color).Render("[" + username + "]")
	return name + " " + text
}

func StyleOwnMessage(text string) string {
	prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#20C997")).Render("You >")
	return prefix + " " + text
}

func StyleSystemMessage(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B8B8B")).Render("✓ " + text)
}

func StyleWarningMessage(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render("⚠ " + text)
}

func StyleErrorMessage(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render("✗ " + text)
}

func StyleHelp() string {
	var b strings.Builder
	b.WriteString("\n")
	cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4DABF7"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CED4DA"))
	b.WriteString(cmdStyle.Render("Commands") + "\n")
	b.WriteString(fmt.Sprintf("  %-12s %s\n", cmdStyle.Render("/help"), descStyle.Render("Show help")))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", cmdStyle.Render("/members"), descStyle.Render("Show online users")))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", cmdStyle.Render("/invite"), descStyle.Render("Invite a user")))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", cmdStyle.Render("/clear"), descStyle.Render("Clear chat")))
	b.WriteString(fmt.Sprintf("  %-12s %s\n", cmdStyle.Render("/exit"), descStyle.Render("Leave room")))
	return b.String()
}

func StyleMembers(members []string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("╔══════════════════════════╗\n")
	title := lipgloss.NewStyle().Bold(true).Render("Online Users")
	b.WriteString(fmt.Sprintf("║ %-22s║\n", title))
	b.WriteString("╠══════════════════════════╣\n")
	if len(members) == 0 {
		b.WriteString(fmt.Sprintf("║  %-20s║\n", "No other users"))
	} else {
		for _, m := range members {
			n := m
			if len(n) > 20 {
				n = n[:20]
			}
			b.WriteString(fmt.Sprintf("║  🟢 %-17s║\n", n))
		}
	}
	b.WriteString("╚══════════════════════════╝\n")
	return b.String()
}

func StyleConnected() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#51CF66")).Render("🟢 Connected")
}

func StyleConnecting() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render("🟡 Connecting...")
}

func StyleDisconnected() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render("🔴 Disconnected")
}
