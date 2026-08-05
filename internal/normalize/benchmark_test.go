package normalize

import "testing"

func BenchmarkNormalize_RuDefault(b *testing.B) {
	norm :=NewNormalizer("ru_default")
	text:= "     ПРИВЕТ, мир! Ёжик   с   яблочком . .    "

	b.ResetTimer()
	for i:=0; i<b.N; i++{
		norm.Normalize(text)
	}
}
func BenchmarkNormalize_Strict(b *testing.B) {
	norm :=NewNormalizer("strict")
	text:= "     ПРИВЕТ, мир! Ёжик   с   яблочком . .    "

	b.ResetTimer()
	for i:=0; i<b.N; i++{
		norm.Normalize(text)
	}
}
func BenchmarkNormalize_RuDefault_10k(b *testing.B) {
	norm :=NewNormalizer("ru_default")
	texts := make([]string, 10000)
	for i:=0; i<10000; i++{
		texts[i] = "     ПРИВЕТ, мир! Ёжик   с   яблочком . .    "
	}
	b.ResetTimer()
	for i:=0; i<b.N; i++{
		for j:=0; j<10000; j++{
		norm.Normalize(texts[j])
		}
	}
}
