package normalize

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type Transform interface {
	Apply(text string) string
}

// Unicode NFC
type NFCTransform struct{}

func (t NFCTransform) Apply(text string) string {
	return norm.NFC.String(text)
}

// Пробелы пр краям
type TrimSpaceTransform struct{}

func (t TrimSpaceTransform) Apply(text string) string {
	return strings.TrimSpace(text)
}

//Замена последовательности пробелов

type CollapseSpacesTransform struct{}

func (t CollapseSpacesTransform) Apply(text string) string {
	spaceRegex := regexp.MustCompile(`\s+`)
	return spaceRegex.ReplaceAllString(text, " ")
}

// приведение к нижнему регистру
type LowerCaseTransform struct{}

func (t LowerCaseTransform) Apply(text string) string {
	return strings.ToLower(text)
}

// замена е на ё
type ReplaceTransform struct{}

func (t ReplaceTransform) Apply(text string) string {
	result := strings.ReplaceAll(text, "ё", "е")
	result = strings.ReplaceAll(result, "Ё", "е")
	return result
}

// Замена пунктуации пробелами
type PunctuationToSpaceTransform struct{}

func (t PunctuationToSpaceTransform) Apply(text string) string {
	result := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return ' '
		}
		return r
	}, text)
	return result
}
