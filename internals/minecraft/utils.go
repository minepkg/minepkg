package minecraft

import (
	"encoding/json"
	"strings"
)

// stringSlice is a slice of strings that can be unmarshalled from a string or a []string
type stringSlice []string

func (w *stringSlice) String() string {
	return strings.Join(*w, " ")
}

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringSlice) UnmarshalJSON(data []byte) (err error) {
	var arg []string

	if string(data[0]) == "[" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		*w = arg
	}

	*w = []string{string(data)}
	return nil
}

// // StringArgument is an argument that can be unmarshalled from an argument object or a string
// type StringArgument struct{ Argument }

// UnmarshalJSON is needed because argument sometimes is a string
func (a *Argument) UnmarshalJSON(data []byte) (err error) {
	var arg Argument

	// looks like an object
	if string(data[0]) == "{" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		return nil
	}

	// looks like a string, wrap it in an argument object
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	// set it as the value
	a.Value = []string{str}
	return nil
}
