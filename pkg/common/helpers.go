package common

import (
	"strings"

	"github.com/google/go-cmp/cmp"
)

type YamlString string

var YamlStringTransformer = cmp.Transformer("YamlLines", func(s YamlString) []string {
	// Splitting string on new line allows diff to be done for each individual
	// line.
	lines := strings.Split(string(s), "\n")

	// Remove and empty lines from the start and end.
	var start, end int
	for i := range lines {
		if lines[i] != "" {
			start = i
			break
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			end = i
			break
		}
	}
	return lines[start : end+1]
})
