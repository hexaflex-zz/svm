package parser

import (
	"strconv"
	"strings"
)

// ParseNumber parses value as an integer.
func ParseNumber(value string) (int64, error) {
	base, value := SplitNumber(value)
	return strconv.ParseInt(value, base, 64)
}

// SplitNumber splits the given number into the base prefix and the
// actual numeric value. Defaults to base-10 if a base-prefix can not
// successfuly be determined. Either there is no prefix, or it is
// not a valid number.
func SplitNumber(v string) (int, string) {
	index := strings.Index(v, "#")
	if index == -1 {
		return 10, v
	}

	base, err := strconv.ParseInt(v[:index], 10, 8)
	if err != nil {
		base = 10
	}

	return int(base), strings.ReplaceAll(v[index+1:], "_", "")
}
