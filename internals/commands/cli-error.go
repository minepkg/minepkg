package commands

import (
	"github.com/charmbracelet/lipgloss"
)

// CliError is an error that might get displayed to the user
type CliError struct {
	Text        string
	Code        string
	Suggestions []string
	Help        string
}

func (e *CliError) Error() string {
	return e.Text
}

func (e *CliError) RichError() string {
	rendered := ErrorBox(e.Text, e.Help)
	if len(e.Suggestions) != 0 {
		suggestionText := "Suggestion:\n"
		if len(e.Suggestions) > 1 {
			suggestionText = "Suggestions:\n"
		}
		suggestionText = Emoji("📎 ") + suggestionText
		for _, s := range e.Suggestions {
			suggestionText += " ⦁ " + s + "\n"
		}
		rendered = lipgloss.JoinVertical(lipgloss.Left, rendered, styleHelpBox.Render(suggestionText))
	}
	return rendered
}
