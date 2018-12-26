package logparser

import (
	"regexp"
	"time"
)

// LogLine is a parsed log line
type LogLine struct {
	Time    time.Time
	Thread  string
	Level   string
	Tag     string
	Message string
	Garbage bool
}

// ParseLine parses a string into a `LogLine`
func ParseLine(input string) *LogLine {
	r := regexp.MustCompile(`\[(\d+:\d+:\d+)\] \[(.+)\/(.+)\] \[(.+)\]: (.+)`)

	found := r.FindStringSubmatch(input)
	if len(found) == 0 {
		return &LogLine{Garbage: true, Message: input}
	}
	time, err := time.Parse("15:04:05", found[1])
	if err != nil {
		return &LogLine{Garbage: true, Message: input}
	}

	parsed := &LogLine{
		Time:    time,
		Thread:  found[2],
		Level:   found[3],
		Tag:     found[4],
		Message: found[5],
	}

	return parsed
}
