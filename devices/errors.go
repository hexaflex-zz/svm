package devices

import "strings"

// ErrorSet defines a list of one or more errors and is itself an error.
type ErrorSet []error

func (e ErrorSet) Len() int {
	return len(e)
}

func (e *ErrorSet) Append(args ...error) {
	*e = append(*e, args...)
}

func (e ErrorSet) Error() string {
	var sb strings.Builder
	for _, err := range e {
		sb.WriteString(err.Error() + "\n")
	}
	return sb.String()
}
