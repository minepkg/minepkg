package utils

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// HumanInteger returns the number in a human readable format
func HumanInteger[N constraints.Integer](input N) string {
	num := uint64(input)
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
