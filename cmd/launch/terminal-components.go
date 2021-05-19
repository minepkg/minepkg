package launch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/pkg/manifest"
)

func dependencyLine(dependency *manifest.DependencyLock) string {
	border := lipgloss.Border{
		Left: "├│",
	}

	version := strings.SplitN(dependency.Version, "-", 2)
	prettyVersion := version[0]
	if len(version) == 2 {
		prettyVersion += gchalk.Gray("-" + version[0])
	}

	paddedName := fmt.Sprintf(" %-25s", dependency.Name)

	line := lipgloss.JoinHorizontal(
		0.5,
		lipgloss.NewStyle().
			Border(border, false, false, false, true).
			Margin(0).
			Padding(0).
			Background(lipgloss.Color("#0e0e0e")).
			MaxWidth(25).
			Width(25).Render(paddedName),
		" "+prettyVersion,
	)
	return line
}
