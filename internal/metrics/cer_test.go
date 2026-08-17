package metrics

import (
	"testing"
)

func TestCER(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		hyp  string
		want CER
	}{
		{
			name: "both_empty",
			ref:  "",
			hyp:  "",
			want: CER{
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
			ref:  "",
			hyp:  "привет",
			want: CER{
				Value:       0.0,
				Distance:    6,
				ManifestLen: 0,
				H:           0,
				S:           0,
				D:           0,
				I:           6,
			},
		},
		{
			name: "empty_hypothesis",
			ref:  "привет",
			hyp:  "",
			want: CER{
				Value:       1.0,
				Distance:    6,
				ManifestLen: 6,
				H:           0,
				S:           0,
				D:           6,
				I:           0,
			},
		},
		{
			name: "one deletion",
			ref:  "мама мыла раму",
			hyp:  "мама раму",
			want: CER{
				Value:       5.0 / 14.0,
				Distance:    5,
				ManifestLen: 14,
				H:           9,
				S:           0,
				D:           5,
				I:           0,
			},
		},
		{
			name: "one insertion",
			ref:  "мама раму",
			hyp:  "мама мыла раму",
			want: CER{
				Value:       5.0 / 9.0,
				Distance:    5,
				ManifestLen: 9,
				H:           9,
				S:           0,
				D:           0,
				I:           5,
			},
		},
		{
			name: "one substitution",
			ref:  "мама мыла раму",
			hyp:  "мама мыла ламу",
			want: CER{
				Value:       1.0 / 14.0,
				Distance:    1,
				ManifestLen: 14,
				H:           13,
				S:           1,
				D:           0,
				I:           0,
			},
		},
		{
			name: "all hit",
			ref:  "мама мылa раму",
			hyp:  "мама мылa раму",
			want: CER{
				Value:       0.0,
				Distance:    0,
				ManifestLen: 14,
				H:           14,
				S:           0,
				D:           0,
				I:           0,
			},
		},
		{
			name: "english",
			ref:  "hello мир",
			hyp:  "hello world",
			want: CER{
				Value:       5.0 / 9.0,
				Distance:    5,
				ManifestLen: 9,
				H:           6,
				S:           3,
				D:           0,
				I:           2,
			},
		},
		{
			name: "mixed",
			ref:  "hello wоrld",
			hyp:  "hello world",
			want: CER{
				Value:       1.0 / 11.0,
				Distance:    1,
				ManifestLen: 11,
				H:           10,
				S:           1,
				D:           0,
				I:           0,
			},
		},
		{
			name: "alignment (ab - ba)",
			ref:  "ab",
			hyp:  "ba",
			want: CER{
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
			ref:  "❤️",
			hyp:  "💙",
			want: CER{
				Value:       1.0,
				Distance:    2,
				ManifestLen: 2,
				H:           0,
				S:           1,
				D:           1,
				I:           0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCER(tt.ref, tt.hyp)

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
