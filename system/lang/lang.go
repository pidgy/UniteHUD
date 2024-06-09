package lang

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	title = cases.Title(language.AmericanEnglish)
)

func Title(s string) string {
	return title.String(s)
}

func Translate(s string) string {
	return s
}
