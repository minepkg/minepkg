package merrors

import (
	"github.com/charmbracelet/lipgloss"
)

// CliError is an error that might get displayed to the user
type CliError struct {
	Text string
	Code string
	Help string
}

var errBox = lipgloss.NewStyle().
	MarginTop(1).
	Bold(true).
	Foreground(lipgloss.Color("#fa8a8a")).
	Background(lipgloss.Color("#512222")).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(lipgloss.Color("#f86262")).
	Padding(1, 2)

var helpBox = lipgloss.NewStyle().
	Foreground(lipgloss.Color("white")).
	// Background(lipgloss.Color("#443030")).
	// Border(lipgloss.NormalBorder(), false, false, false, true).
	// BorderLeftForeground(lipgloss.Color("white")).
	Padding(0, 2).
	Margin(0, 1)

func (e *CliError) Error() string {
	str := errBox.Render("❗ Error: "+e.Text) + "\n"
	if e.Help != "" {
		str += helpBox.Render("🔹 Suggestion:\n" + e.Help)
	}
	return str
}
