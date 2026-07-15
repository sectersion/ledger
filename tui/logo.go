package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// sidebarContentWidth fits the "LEDGER" block-letter logo (35 cols) with a
// little breathing room, when the terminal is wide enough for it — see
// Model.sidebarContentWidth for the narrow-terminal fallback.
const sidebarContentWidth = 36

var logoGlyphs = map[byte][5]string{
	'L': {"█    ", "█    ", "█    ", "█    ", "█████"},
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'D': {"████ ", "█   █", "█   █", "█   █", "████ "},
	'G': {" ████", "█    ", "█  ██", "█   █", " ████"},
	'R': {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
}

// renderLogo draws "LEDGER" as 5-row block ASCII art with a left-to-right
// gradient from the theme's primary color to its accent color.
func renderLogo() string {
	const word = "LEDGER"
	var rows [5]strings.Builder
	for i := 0; i < len(word); i++ {
		glyph := logoGlyphs[word[i]]
		for r := 0; r < 5; r++ {
			rows[r].WriteString(glyph[r])
			if i < len(word)-1 {
				rows[r].WriteByte(' ')
			}
		}
	}
	width := rows[0].Len()

	var out strings.Builder
	for r := 0; r < 5; r++ {
		line := rows[r].String()
		for x, ch := range line {
			if ch == ' ' {
				out.WriteByte(' ')
				continue
			}
			t := 0.0
			if width > 1 {
				t = float64(x) / float64(width-1)
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(lerpHex(string(colorPrimary), string(colorAccent), t)))
			out.WriteString(style.Render(string(ch)))
		}
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n")
}

func lerpHex(a, b string, t float64) string {
	ar, ag, ab := hexRGB(a)
	br, bg, bb := hexRGB(b)
	r := int(float64(ar) + t*float64(br-ar))
	g := int(float64(ag) + t*float64(bg-ag))
	bl := int(float64(ab) + t*float64(bb-ab))
	return fmt.Sprintf("#%02X%02X%02X", r, g, bl)
}

func hexRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	r, _ := strconv.ParseInt(hex[0:2], 16, 32)
	g, _ := strconv.ParseInt(hex[2:4], 16, 32)
	b, _ := strconv.ParseInt(hex[4:6], 16, 32)
	return int(r), int(g), int(b)
}
