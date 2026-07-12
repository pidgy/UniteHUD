package lang

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	Titled = cases.Title(language.AmericanEnglish)
	upper  = cases.Upper(language.AmericanEnglish)
)

func Title(s string) string {
	return Titled.String(s)
}

// TitleFirst uppercases the first rune of a string.
func TitleFirst(s string) string {
	if len(s) < 1 {
		return ""
	}
	return upper.String(s[:1]) + s[1:]
}

func Translate(s string) string {
	return s
}

func Upper(s string) string {
	return upper.String((s))
}
