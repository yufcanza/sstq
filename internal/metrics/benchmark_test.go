package metrics

import "testing"

func BenchmarkWER(b *testing.B) {
	ref := []string{"мама", "мыла", "раму", "в", "ванной", "комнате"}
	hyp := []string{"мама", "мыла", "раму", "в", "ванной", "комнате"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateWER(ref, hyp)
	}
}
func BenchmarkCER(b *testing.B) {
	ref := "мама мыла раму в ванной комнате"
	hyp := "мама мыла раму в ванной комнате"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateCER(ref, hyp)
	}
}
func BenchmarkWER_10k(b *testing.B) {
	refs := make([][]string, 10000)
	hyps := make([][]string, 10000)
	for i := 0; i < 10000; i++ {
		refs[i] = []string{"мама", "мыла", "раму", "в", "ванной", "комнате"}
		hyps[i] = []string{"мама", "мыла", "раму", "в", "ванной", "комнате"}

	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			CalculateWER(refs[j], hyps[j])
		}
	}
}
func BenchmarkCER_10k(b *testing.B) {
	refs := make([]string, 10000)
	hyps := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		refs[i] = "мама мыла раму в ванной комнате"
		hyps[i] = "мама мыла раму в ванной комнате"

	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			CalculateCER(refs[j], hyps[j])
		}
	}
}
