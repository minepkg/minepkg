package minecraft

import (
	"encoding/json"
	"testing"
)

func TestStringSlice(t *testing.T) {
	var s stringSlice
	err := json.Unmarshal([]byte(`["a", "b"]`), &s)
	if err != nil {
		t.Fatal(err)
	}
	if s.String() != "a b" {
		t.Fatalf("Expected 'a b', got '%s'", s.String())
	}

	err = json.Unmarshal([]byte(`"a b"`), &s)
	if err != nil {
		t.Fatal(err)
	}
	if s.String() != "a b" {
		t.Fatalf("Expected 'a b', got '%s'", s.String())
	}
}