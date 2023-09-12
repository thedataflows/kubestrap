package stringutil

import "strings"

// ConcatStrings returns concatenated strings
func ConcatStrings(strs ...string) string {
	var strBuilder strings.Builder
	for _, s := range strs {
		strBuilder.WriteString(s)
	}
	return strBuilder.String()
}
