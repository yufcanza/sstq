package normalize

import (
	"strings"

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
