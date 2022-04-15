package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var styleErrBox = lipgloss.NewStyle().
	MarginTop(1).
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#b71c1c", Dark: "#fa8a8a"}).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(lipgloss.Color("#f86262")).
	Padding(0, 1)

var styleHelpBox = lipgloss.NewStyle().
	Padding(0, 2).
	Margin(0, 1).
	PaddingTop(1)

var grassBorder = lipgloss.Border{
	// Top: "▪",
	Top: "▂",
}

var StyleGrass = lipgloss.NewStyle().
	Background(lipgloss.Color("#7a563b")).
	Border(grassBorder, true, false, false, false).
	BorderForeground(lipgloss.Color("#63a73c")).
	Padding(0, 2)

var styleErrText = lipgloss.NewStyle().Width(62)

var warnBorder = lipgloss.Border{
	Top: "▪",
}

var StyleWarnBox = lipgloss.NewStyle().
	MarginTop(1).
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#774d00", Dark: "#ffe5d5"}).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(lipgloss.Color("#ffb837")).
	Padding(0, 1)

var StyleInfoBox = lipgloss.NewStyle().
	MarginTop(1).
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#00174c", Dark: "#e7d5ff"}).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(lipgloss.Color("#5e5eff")).
	Padding(1, 1)

func ErrorBox(errorString string, helpText string) string {
	rendered := styleErrBox.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top, Emoji("💥 "),
			styleErrText.Render(fmt.Sprintf("Error: %s", errorString)),
		),
	)
	if helpText != "" {
		rendered += styleErrBox.Render(fmt.Sprint(Emoji("❔ "), helpText))
	}

	return rendered
}
