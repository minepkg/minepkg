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
		return nil
	}

	*w = []string{string(data)}
	return nil
}
