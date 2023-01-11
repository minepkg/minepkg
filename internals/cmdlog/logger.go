package cmdlog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jwalton/gchalk"
)

var DefaultLogger = New()

// Logger logs pretty stuff to the console
type Logger struct {
	emojis    bool
	indention int
}

// helper for indention
func (l *Logger) println(a string) {
	fmt.Println(strings.Repeat(" ", l.indention) + a)
}

// printEmoji prints string e only when emojis are enabled
func (l *Logger) printEmoji(e string) {
	if l.emojis {
		fmt.Print(e + " ")
	}
}

func (l *Logger) sprintEmoji(e string) string {
	if l.emojis {
		return e
	}
	return ""
}

// Headline prints a blue line
func (l *Logger) Headline(s string) {
	fmt.Println(gchalk.WithCyan().Bold(s))
}

// Info prints a "normal" line
func (l *Logger) Info(s string) {
	l.println(s)
}

// Log prints a black line
func (l *Logger) Log(s string) {
	fmt.Println(gchalk.Gray(s))
}

// Warn will print a warning
func (l *Logger) Warn(s ...string) {
	l.printEmoji("⚠️ ")
	lines := make([]string, len(s))
	for i, line := range s {
		lines[i] = gchalk.WithYellow().Bold(line)
	}
	fmt.Println(strings.Join(lines, " "))
}

// Fail will print the given message with PrintLn and then exit 1
func (l *Logger) Fail(s string) {
	l.printEmoji("💣")
	fmt.Print(gchalk.WithRed().Bold("Error: "))
	fmt.Println(gchalk.WithWhite().Bold(s))
	os.Exit(1)
}

// NewTask returns a new Task logger
func (l *Logger) NewTask(end int) *Task {
	logger := *l
	task := Task{&logger, 0, end, time.Now()}
	// TODO:
	// task.indention = 2
	return &task
}

// New returns a new Logger
func New() *Logger {
	emojis := runtime.GOOS != "windows"

	// disable color for CI
	if os.Getenv("CI") != "" {
		emojis = false
	}
	return &Logger{emojis: emojis}
}

// Task logs but with progress
type Task struct {
	*Logger
	current   int
	end       int
	startTime time.Time
}

// Step prints progress
func (l *Task) Step(e string, s string) {
	if l.current > 0 {
		elapsed := time.Since(l.startTime)
		fmt.Printf(" took %s\n", elapsed)
	}
	l.current++
	text := fmt.Sprintf(
		"[%d / %d] %s %s",
		l.current,
		l.end,
		l.sprintEmoji(e),
		s,
	)

	fmt.Println(gchalk.Cyan(text))
}

// HumanUint32 returns the number in a human readable format
func HumanUint32(num uint32) string {
	switch {
	case num >= 1000000000:
		return fmt.Sprintf("%v B", num/1000000000)
	case num >= 1000000:
		return fmt.Sprintf("%v M", num/1000000)
	case num >= 1000:
		return fmt.Sprintf("%v K", num/1000)
	}
	return fmt.Sprintf("%v", num)
}

// HumanFloat32 returns the number in a human readable format
func HumanFloat32(num float32) string {
	switch {
	case num >= 1000000000:
		return fmt.Sprintf("%v B", num/1000000000)
	case num >= 1000000:
		return fmt.Sprintf("%v M", num/1000000)
	case num >= 1000:
		return fmt.Sprintf("%v K", num/1000)
	}
	return fmt.Sprintf("%v", num)
}
