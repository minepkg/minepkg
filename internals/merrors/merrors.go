package merrors

import "fmt"

// CliError is an error that might get displayed to the user
type CliError struct {
	Text string
	Code string
	Help string
}

func (e *CliError) Error() string {
	str := fmt.Sprintf("%s\n", e.Text)
	if e.Help != "" {
		str += "\n  Help: " + e.Help
	}
	return str
}
