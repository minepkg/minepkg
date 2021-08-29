// mostly stolen from https://gist.github.com/crgimenes/c3b8b4fcce8529e9201f83c8da134f32
package fabric

import (
	"encoding/json"
	"errors"
)

var (
	// ErrUnsupportedType is returned if the type is not implemented
	ErrUnsupportedType = errors.New("unsupported type")
)

// StrArray string array to be used on JSON UnmarshalJSON
type StrArray []string

// UnmarshalJSON convert JSON object array of string or
// a string format strings to a golang string array
func (sa *StrArray) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	// it's a string
	case string:
		*sa = StrArray([]string{obj})
		return nil
	// it's an array of strings
	case []interface{}:
		s := make([]string, 0, len(obj))
		// check if it really are only strings
		for _, v := range obj {
			value, ok := v.(string)
			if !ok {
				return ErrUnsupportedType
			}
			s = append(s, value)
		}
		*sa = StrArray(s)
		return nil
	}
	return ErrUnsupportedType
}

// MarshalJSON convert a golang string array to a JSON object array of string or a string
func (sa StrArray) MarshalJSON() ([]byte, error) {
	if len(sa) == 1 {
		return json.Marshal(sa[0])
	}
	return json.Marshal([]string(sa))
}
