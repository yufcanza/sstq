package metrics

import (
	"testing"
)

func TestWER(t *testing.T) {
	tests := []struct {
		name string
		ref  []string
		hyp  []string
		want WER
	}{
		{
			name: "both_empty",
			ref:  []string{},
			hyp:  []string{},
			want: WER{
				Value:       0.0,
				Distance:    0,
				ManifestLen: 0,
				H:           0,
				S:           0,
				D:           0,
				I:           0,
			},
		},
		{
			name: "empty_reference",
			ref:  []string{},
			hyp:  []string{"привет"},
			want: WER{
				Value:       0.0,
				Distance:    1,
				ManifestLen: 0,
				H:           0,
				S:           0,
				D:           0,
				I:           1,
			},
		},
		{
			name: "empty_hypothesis",
			ref:  []string{"привет"},
			hyp:  []string{},
			want: WER{
				Value:       1.0,
				Distance:    1,
				ManifestLen: 1,
				H:           0,
				S:           0,
				D:           1,
				I:           0,
			},
		},
		{
			name: "one deletion",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "раму"},
			want: WER{
				Value:       1.0 / 3.0,
				Distance:    1,
				ManifestLen: 3,
				H:           2,
				S:           0,
				D:           1,
				I:           0,
			},
		},
		{
			name: "one insertion",
			ref:  []string{"мама", "раму"},
			hyp:  []string{"мама", "мыла", "раму"},
			want: WER{
				Value:       1.0 / 2.0,
				Distance:    1,
				ManifestLen: 2,
				H:           2,
				S:           0,
				D:           0,
				I:           1,
			},
		},
		{
			name: "one substitution",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "мыла", "ламу"},
			want: WER{
				Value:       1.0 / 3.0,
				Distance:    1,
				ManifestLen: 3,
				H:           2,
				S:           1,
				D:           0,
				I:           0,
			},
		},
		{
			name: "all hit",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "мыла", "раму"},
			want: WER{
				Value:       0.0,
				Distance:    0,
				ManifestLen: 3,
				H:           3,
				S:           0,
				D:           0,
				I:           0,
			},
		},
		{
			name: "english",
			ref:  []string{"hello", "мир"},
			hyp:  []string{"hello", "world"},
			want: WER{
				Value:       1.0 / 2.0,
				Distance:    1,
				ManifestLen: 2,
				H:           1,
				S:           1,
				D:           0,
				I:           0,
			},
		},
		{
			name: "mixed",
			ref:  []string{"hello", "wоrld"},
			hyp:  []string{"hello", "world"},
			want: WER{
				Value:       1.0 / 2.0,
				Distance:    1,
				ManifestLen: 2,
				H:           1,
				S:           1,
				D:           0,
				I:           0,
			},
		},
		{
			name: "(ab - ba)",
			ref:  []string{"a", "b"},
			hyp:  []string{"b", "a"},
			want: WER{
				Value:       1.0,
				Distance:    2,
				ManifestLen: 2,
				H:           0,
				S:           2,
				D:           0,
				I:           0,
			},
		},
		{
			name: "emoji",
			ref:  []string{"Привет", "❤️", "мир"},
			hyp:  []string{"Привет", "💙", "мир"},
			want: WER{
				Value:       1.0 / 3.0,
				Distance:    1,
				ManifestLen: 3,
				H:           2,
				S:           1,
				D:           0,
				I:           0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateWER(tt.ref, tt.hyp)

			if result.Value != tt.want.Value {
				t.Errorf("Value = %v, want %v", result.Value, tt.want.Value)
			}
			if result.Distance != tt.want.Distance {
				t.Errorf("Distance = %v, want %v", result.Distance, tt.want.Distance)
			}
			if float64(result.ManifestLen) != float64(tt.want.ManifestLen) {
				t.Errorf("Len = %v, want %v", result.ManifestLen, tt.want.ManifestLen)
			}
			if result.H != tt.want.H {
				t.Errorf("Hits = %v, want %v", result.H, tt.want.H)
			}
			if result.S != tt.want.S {
				t.Errorf("Substitution = %v, want %v", result.S, tt.want.S)
			}
			if result.D != tt.want.D {
				t.Errorf("Deletion = %v, want %v", result.D, tt.want.D)
			}
			if result.I != tt.want.I {
				t.Errorf("Insertion = %v, want %v", result.I, tt.want.I)
			}

		})
	}
}
