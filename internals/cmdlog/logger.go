package cmdlog

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gookit/color"
)

// Logger loggs pretty stuff to the console
type Logger struct {
	emojis    bool
	color     bool
	indention int
}

// helper for indention
func (l *Logger) println(a string) {
	fmt.Println(strings.Repeat(" ", l.indention) + a)
}

// pritEmoji prints string e only when emojis are enabled
func (l *Logger) pritEmoji(e string) {
	if l.emojis == true {
		fmt.Print(e + " ")
	}
}

func (l *Logger) sprintEmoji(e string) string {
	if l.emojis == true {
		return e
	}
	return ""
}

// Headline prints a blue line
func (l *Logger) Headline(s string) {
	color.Style{color.FgCyan, color.OpBold}.Println(s)
}

// Info prints a "normal" line
func (l *Logger) Info(s string) {
	l.println(s)
}

// Log prints a black line
func (l *Logger) Log(s string) {
	color.LightWhite.Println(s)
}

// Warn will print a warning
func (l *Logger) Warn(s string) {
	l.pritEmoji("⚠️ ")
	color.Style{color.FgYellow, color.OpBold}.Println(s)
}

// Fail will print the given message with PrintLn and then exit 1
func (l *Logger) Fail(s string) {
	l.pritEmoji("💣")
	color.Style{color.FgRed, color.OpBold}.Print("Error: ")
	color.Style{color.FgWhite, color.OpBold}.Println(s)
	os.Exit(1)
}

// NewTask returns a new Task logger
func (l *Logger) NewTask(end int) *Task {
	logger := *l
	task := Task{&logger, 0, end}
	// TODO:
	// task.indention = 2
	return &task
}

// New returns a new Logger
func New() *Logger {
	emojis := runtime.GOOS != "windows"
	colorToggle := true

	// disable color for CI
	if os.Getenv("CI") != "" {
		emojis = false
		colorToggle = false
		color.Disable()
	}
	return &Logger{emojis: emojis, color: colorToggle}
}

// Task logs but with progress
type Task struct {
	*Logger
	current int
	end     int
}

// Step prints progress
func (l *Task) Step(e string, s string) {
	l.current++
	text := color.Cyan.Sprintf(
		"[%d / %d] %s %s",
		l.current,
		l.end,
		l.sprintEmoji(e),
		s,
	)

	// we don't use l.println here, because step headlines should have no indentation
	fmt.Println(text)
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

// Fail will print the given message with PrintLn and then exit 1
func Fail(a ...interface{}) {
	fmt.Println(a...)
	os.Exit(1)
}

// Failf will print the given message with Printf and then exit 1
func Failf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	os.Exit(1)
}
