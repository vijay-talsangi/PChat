package chat

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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

type connState int

const (
	StateConnecting connState = iota
	StateConnected
	StateDisconnected
	StateFailed
)

func stateBadge(state connState) string {
	var bg, label string
	switch state {
	case StateConnected:
		bg = "#2D8A4E"
		label = "Connected"
	case StateConnecting:
		bg = "#B8860B"
		label = "Connecting"
	case StateDisconnected, StateFailed:
		bg = "#C92A2A"
		label = "Disconnected"
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 2).
		Render(label)
}

func RenderHeader(width int, roomName, username string, state connState) string {
	if width < 20 {
		width = 20
	}
	boxW := width - 6
	innerW := boxW - 2
	badge := stateBadge(state)
	titleLeft := "  🔒 PChat  "
	titleRight := "  " + badge + "  "
	titlePad := innerW - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if titlePad < 1 {
		titlePad = 1
	}
	titleLine := titleLeft + strings.Repeat(" ", titlePad) + titleRight
	content := fmt.Sprintf("%s\n  Room: %s\n  User: %s", titleLine, roomName, username)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4DABF7")).
		Width(boxW).
		Padding(0, 1).
		Render(content)
	centered := lipgloss.Place(width, 5, lipgloss.Center, lipgloss.Top, box)
	return "\n" + centered + "\n"
}

func formatTimestamp(t time.Time) string {
	return t.Format("3:04 PM")
}

func StylePeerMessage(username, text, timestamp string) string {
	col := hashColor(username)
	name := lipgloss.NewStyle().Bold(true).Foreground(col).Render("[" + username + "]")
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	left := name + "  " + text
	return left + ts
}

func StyleGroupedMessage(indentWidth int, text, timestamp string) string {
	indent := strings.Repeat(" ", indentWidth)
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	left := indent + text
	return left + ts
}

func StyleOwnMessage(text, timestamp string) string {
	prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#20C997")).Render("You >")
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	left := prefix + "  " + text
	return left + ts
}

func StyleSystemMessage(text, timestamp string) string {
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B8B8B")).Render("✓ "+text) + ts
}

func StyleWarningMessage(text, timestamp string) string {
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD43B")).Render("⚠ "+text) + ts
}

func StyleErrorMessage(text, timestamp string) string {
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C757D")).Render("  " + timestamp)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render("✗ "+text) + ts
}

func StyleHelp() string {
	var b strings.Builder
	b.WriteString("\n")
	cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4DABF7"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CED4DA"))
	b.WriteString(cmdStyle.Render("Commands"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %s%s %s\n", cmdStyle.Render("/help"), strings.Repeat(" ", 12-len("/help")), descStyle.Render("Show help"))
	fmt.Fprintf(&b, "  %s%s %s\n", cmdStyle.Render("/members"), strings.Repeat(" ", 12-len("/members")), descStyle.Render("Show online users"))
	fmt.Fprintf(&b, "  %s%s %s\n", cmdStyle.Render("/invite"), strings.Repeat(" ", 12-len("/invite")), descStyle.Render("Invite a user"))
	fmt.Fprintf(&b, "  %s%s %s\n", cmdStyle.Render("/clear"), strings.Repeat(" ", 12-len("/clear")), descStyle.Render("Clear chat"))
	fmt.Fprintf(&b, "  %s%s %s\n", cmdStyle.Render("/exit"), strings.Repeat(" ", 12-len("/exit")), descStyle.Render("Leave room"))
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

func RenderUnreadIndicator(count int) string {
	s := fmt.Sprintf("↓ %d new message(s)", count)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4DABF7")).
		Bold(true).
		Render(s)
}

func RenderInput(inputView string, width int, focused bool) string {
	borderColor := lipgloss.Color("#495057")
	if focused {
		borderColor = lipgloss.Color("#4DABF7")
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Padding(0, 1).
		Render(inputView)
}
