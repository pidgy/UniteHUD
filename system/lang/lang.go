package lang

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	Titled = cases.Title(language.AmericanEnglish)
)

func Title(s string) string {
	return Titled.String(s)
}

func Translate(s string) string {
	return s
}
