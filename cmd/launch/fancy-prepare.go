package launch

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/gchalk"
	"github.com/minepkg/minepkg/internals/resolver"
	"github.com/minepkg/minepkg/internals/resolver/providers"
)

type FancyResolveUI struct {
	resolver    *resolver.Resolver
	items       []providers.Result
	doneC       chan bool
	errorC      chan error
	resolved    uint32
	transferred uint64
	total       uint64
}

func NewResolverUI(resolver *resolver.Resolver) *FancyResolveUI {
	return &FancyResolveUI{
		resolver: resolver,
		items:    []providers.Result{},
		doneC:    make(chan bool),
		errorC:   make(chan error),
	}
}

// func (m *FancyResolveUI) receiveResult() tea.Msg {
// 	result, more := <-m.resultsChan
// 	if !more {
// 		return false
// 	}
// 	return result
// }

func (m *FancyResolveUI) check() tea.Msg {
	select {
	case err := <-m.errorC:
		fmt.Println("dang")
		fmt.Println(err)
		return false
	case <-m.doneC:
		return false
	default:
		time.Sleep(time.Second / 10)
		return nil
	}
}

func (m FancyResolveUI) Init() tea.Cmd {
	go func() {
		err := m.resolver.Resolve(context.TODO())
		if err != nil {
			m.errorC <- err
			return
		}
		m.doneC <- true
	}()
	return m.check
}

func (m FancyResolveUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	transferred := uint64(0)
	total := uint64(100)
	resolved := uint32(0)

	for _, item := range m.resolver.BetterResolved {
		transferred += item.Transferred()
		total += item.Size()
		if item.Result != nil {
			resolved += 1
		}
	}

	m.transferred = transferred
	m.total = total
	m.resolved = resolved

	switch msg := msg.(type) {

	// case providers.Result:
	// 	m.items = append(m.items, msg)
	// 	return m, m.receiveResult
	case bool:
		return m, tea.Quit
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit
		}

	}

	// Return the updated FancyResolveUI to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, m.check
}

func (m FancyResolveUI) View() string {
	if !m.resolver.ResolveFinished() {
		fmt.Println("test")
		return fmt.Sprintf("┃ resolving (%d)\n", m.resolved)
	}

	return fmt.Sprintf(
		"┃ Downloading %d / %d MiB",
		m.transferred/(1024*1024),
		m.total/(1024*1024),
	)
}

func old(item resolver.Resolved, rows []string) {
	// lock := item.Result.Lock()
	result := item.Result

	border := lipgloss.Border{
		Left: "├│",
	}

	version := strings.SplitN(result.Lock().Version, "-", 2)
	prettyVersion := version[0]
	if len(version) == 2 {
		prettyVersion += gchalk.Gray("-" + version[1])
	}

	paddedName := fmt.Sprintf(" %-25s", result.Lock().Name)
	loadingText := strings.Builder{}
	progressPos := int(item.Progress() * 25)

	// good: #034497
	// also good: #0f3c76
	loadingText.Write([]byte(gchalk.BgHex("#11593c")(paddedName[:progressPos])))
	loadingText.Write([]byte(gchalk.BgHex("#000")(paddedName[progressPos:])))

	line := lipgloss.JoinHorizontal(
		0.5,
		lipgloss.NewStyle().
			Border(border, false, false, false, true).
			Margin(0).
			Padding(0).
			Background(lipgloss.Color("#0e0e0e")).
			MaxWidth(25).
			Width(25).Render(loadingText.String()),
		" "+prettyVersion,
	)
	rows = append(rows, line)
}
