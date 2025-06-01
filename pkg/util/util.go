package util

import (
	"regexp"
	"strings"
)

var thinkRegex = regexp.MustCompile(`(?s)^<think>.*?</think>`)

func RemoveThinkTag(input string) string {
	if !strings.HasPrefix(input, "<think>") {
		return input
	}

	return strings.TrimSpace(thinkRegex.ReplaceAllString(input, ""))
}
