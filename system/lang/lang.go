package lang

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

func Translate(s string) string {
	return s
}

func Upper(s string) string {
	return upper.String((s))
}
