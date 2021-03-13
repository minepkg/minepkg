package logparser

import (
	"fmt"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{
			name: "crap",
			arg:  "I am crap string",
		},
		{
			name: "test",
			arg:  "[13:46:33] [main/INFO] [FML]: Forge bla bla for Minecraft 1.12.2 loading",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := ParseLine(tt.arg)
			if line.String() != tt.arg {
				fmt.Printf("%d vs %d", len(line.String()), len(tt.arg))
				t.Fatalf("Input \"%s\" did not produce same output: \nexpected %s\ngot      %s\n", tt.name, tt.arg, line.String())
			}
		})
	}
}
