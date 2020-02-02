package merrors

import "fmt"

// CliError is a error that might get displayed to the user
type CliError struct {
	Err  string
	Code string
	Help string
}

func (e *CliError) Error() string {
	str := fmt.Sprintf("%s\n", e.Err)
	if e.Help != "" {
		str += "\n  Help: " + e.Help
	}
	return str
}
