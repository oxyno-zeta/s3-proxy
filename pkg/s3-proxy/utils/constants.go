package utils

import "regexp"

// NewLineMatcherRegex Regex to remove all new lines.
var NewLineMatcherRegex = regexp.MustCompile(`\r?\n`)
