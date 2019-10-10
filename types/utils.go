package types

import (
	"regexp"
	"strings"
)

const filterPattern = `[^a-z0-9]+`

var disabledSymbols = regexp.MustCompile(filterPattern)

func CleanString(str string) string {
	return disabledSymbols.ReplaceAllString(strings.ToLower(str), "")
}
