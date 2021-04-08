package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var styleErrBox = lipgloss.NewStyle().
	Width(80).
	MarginTop(1).
	Bold(true).
	Background(lipgloss.AdaptiveColor{Light: "#ffcdd2", Dark: "#512222"}).
	Foreground(lipgloss.AdaptiveColor{Light: "#b71c1c", Dark: "#fa8a8a"}).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(lipgloss.Color("#f86262")).
	Padding(1, 2)

var styleHelpBox = lipgloss.NewStyle().
	Width(80).
	Background(lipgloss.AdaptiveColor{Light: "#e9e9e9", Dark: "#2f2f2f"}).
	Padding(0, 2).
	Margin(0, 1).
	PaddingTop(1)

var styleErrText = lipgloss.NewStyle().Width(62)

func ErrorBox(errorString string, helpText string) string {
	rendered := styleErrBox.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top, Emoji("❗ "),
			styleErrText.Render(fmt.Sprintf("Error: %s", errorString)),
		),
	)
	if helpText != "" {
		rendered += styleErrBox.Render(fmt.Sprintf("%sError: %s", Emoji("❔ "), errorString))
	}

	return rendered
}
