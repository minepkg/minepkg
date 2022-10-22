package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/minepkg/minepkg/internals/instances"
	"github.com/minepkg/minepkg/internals/provider"
)

type OutdatedChecker struct {
	Context  context.Context
	Instance *instances.Instance
	toCheck  uint32
	result   []instances.OutdatedResult
}

func (o *OutdatedChecker) Check() error {
	dependencies := o.Instance.GetDependencyList()

	// thread safe list of outdated dependencies
	outdated := make(chan *instances.OutdatedResult, len(dependencies))
	o.toCheck = uint32(len(dependencies))

	fmt.Println("starting check")

	// wg := sync.WaitGroup{}
	// wg.Add(len(dependencies))

	cErr := make(chan error, 1)
	for _, dependency := range dependencies {
		go func(dependency instances.Dependency) {
			// defer wg.Done()
			result, err := o.Instance.ProviderStore.ResolveLatest(o.Context, dependency.ProviderRequest())
			if err != nil {
				cErr <- err
				return
			}
			outdated <- &instances.OutdatedResult{
				Dependency: dependency,
				Result:     result,
			}

		}(dependency)
	}

	// build list
	o.result = make([]instances.OutdatedResult, 0, len(dependencies))
	for range dependencies {
		select {
		case err := <-cErr:
			if errors.Is(err, provider.ErrProviderUnsupported) {
				o.result = append(o.result, instances.OutdatedResult{})
				continue
			}
			return err
		case result := <-outdated:
			o.result = append(o.result, instances.OutdatedResult{
				Dependency: result.Dependency,
				Result:     result.Result,
			})
		}
	}

	return nil
}

func (o *OutdatedChecker) ProgressValue() int {
	return len(o.result)
}

// ProgressPercent returns the current progress of the check in percent
func (o *OutdatedChecker) ProgressPercent() float64 {
	return float64(len(o.result)) / float64(o.toCheck)
}

// Total returns the total number of packages to check
func (o *OutdatedChecker) Total() int {
	return int(o.toCheck)
}

type model struct {
	checker  *OutdatedChecker
	width    int
	height   int
	spinner  spinner.Model
	progress progress.Model
	done     bool
}

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	subtleStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

func newModel(checker *OutdatedChecker) model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	p.Full = '－'
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return model{
		checker:  checker,
		spinner:  s,
		progress: p,
	}
}

func (m model) Init() tea.Cmd {
	go m.checker.Check()
	return tea.Batch(spinner.Tick, m.check())
}

func (m model) check() tea.Cmd {
	d := time.Millisecond * 250
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return checkMsg("bla")
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	progressPercent := m.checker.ProgressPercent()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	case checkMsg:
		// Update progress bar
		progressCmd := m.progress.SetPercent(progressPercent)
		if progressPercent == 1 {
			// Everything's been installed. We're done!
			m.done = true
			return m, tea.Quit
		}

		return m, tea.Batch(
			m.check(),
			progressCmd,
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	total := m.checker.Total()
	progressValue := m.checker.ProgressValue()
	w := lipgloss.Width(fmt.Sprintf("%d", total))

	if m.done {
		return doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", total))
	}

	pkgCount := fmt.Sprintf(" %*d/%*d", w, progressValue, w, total)

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	// cellsAvail := max(0, m.width-lipgloss.Width(spin+prog+pkgCount))

	// pkgName := currentPkgNameStyle.Render(m.packages[m.index])
	// info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Installing " + pkgName)

	cellsRemaining := max(0, m.width-lipgloss.Width(spin+prog+pkgCount))
	gap := strings.Repeat(" ", cellsRemaining)

	return spin + gap + prog + pkgCount
}

type checkMsg string

// func downloadAndInstall(pkg string) tea.Cmd {
// 	// This is where you'd do i/o stuff to download and install packages. In
// 	// our case we're just pausing for a moment to simulate the process.
// 	d := time.Millisecond * time.Duration(rand.Intn(500))
// 	return tea.Tick(d, func(t time.Time) tea.Msg {
// 		return installedPkgMsg(pkg)
// 	})
// }

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
