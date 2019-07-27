package minecraft

import (
	"encoding/json"
	"strings"
)

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

type stringArgument struct{ argument }

// UnmarshalJSON is needed because argument sometimes is a string
func (w *stringArgument) UnmarshalJSON(data []byte) (err error) {
	var arg argument
	if string(data[0]) == "{" {
		err := json.Unmarshal(data, &arg)
		if err != nil {
			return err
		}
		w.argument = arg
		return nil
	}

	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	w.Value = []string{str}
	return nil
}
