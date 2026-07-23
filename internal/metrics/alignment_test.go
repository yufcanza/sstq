package metrics

import (
	"testing"
)

func TestAlignment(t *testing.T) {
	tests := []struct {
		name string
		ref  []string
		hyp  []string
		want []AlignmentItem
	}{

		{
			name: "both_empty",
			ref:  []string{},
			hyp:  []string{},
			want: []AlignmentItem{},
		},
		{
			name: "empty_reference",
			ref:  []string{},
			hyp:  []string{"привет"},
			want: []AlignmentItem{
				{Type: "insert", Manifest: "", Hypothesis: "привет"},
			},
		},
		{
			name: "empty_hypothesis",
			ref:  []string{"привет"},
			hyp:  []string{},
			want: []AlignmentItem{
				{Type: "delete", Manifest: "привет", Hypothesis: ""},
			},
		},
		{
			name: "one deletion",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "раму"},
			want: []AlignmentItem{
				{Type: "equal", Manifest: "мама", Hypothesis: "мама"},
				{Type: "delete", Manifest: "мыла", Hypothesis: ""},
				{Type: "equal", Manifest: "раму", Hypothesis: "раму"},
			},
		},
		{
			name: "one insertion",
			ref:  []string{"мама", "раму"},
			hyp:  []string{"мама", "мыла", "раму"},
			want: []AlignmentItem{
				{Type: "equal", Manifest: "мама", Hypothesis: "мама"},
				{Type: "insert", Manifest: "", Hypothesis: "мыла"},
				{Type: "equal", Manifest: "раму", Hypothesis: "раму"},
			},
		},
		{
			name: "one substitution",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "мыла", "ламу"},
			want: []AlignmentItem{
				{Type: "equal", Manifest: "мама", Hypothesis: "мама"},
				{Type: "equal", Manifest: "мыла", Hypothesis: "мыла"},
				{Type: "substitute", Manifest: "раму", Hypothesis: "ламу"},
			},
		},
		{
			name: "all hit",
			ref:  []string{"мама", "мыла", "раму"},
			hyp:  []string{"мама", "мыла", "раму"},
			want: []AlignmentItem{
				{Type: "equal", Manifest: "мама", Hypothesis: "мама"},
				{Type: "equal", Manifest: "мыла", Hypothesis: "мыла"},
				{Type: "equal", Manifest: "раму", Hypothesis: "раму"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateAlignment(tt.ref, tt.hyp)

			if len(result.Items) != len(tt.want) {
				t.Errorf("Items length = %v, expected %v", len(result.Items), len(tt.want))
				return
			}

			for i, item := range result.Items {
				if item.Type != tt.want[i].Type {
					t.Errorf("Item %d Type = %v, expected %v", i, item.Type, tt.want[i].Type)
				}
				if item.Manifest != tt.want[i].Manifest {
					t.Errorf("Item %d Reference = %v, expected %v", i, item.Manifest, tt.want[i].Manifest)
				}
				if item.Hypothesis != tt.want[i].Hypothesis {
					t.Errorf("Item %d Hypothesis = %v, expected %v", i, item.Hypothesis, tt.want[i].Hypothesis)
				}

			}
		})
	}

}
