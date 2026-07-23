package normalize

import (
	"testing"
)

func TestStrictProfile(t *testing.T) {
	norm := NewNormalizer("strict")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Регистр",
			input: "Привет Мир",
			want:  "Привет Мир",
		},
		{
			name:  "Пунктуация",
			input: "Привет, Мир!",
			want:  "Привет, Мир!",
		},
		{
			name:  "Пробелы внутри",
			input: "Привет        Мир",
			want:  "Привет Мир",
		},
		{
			name:  "Пробелы снаружи",
			input: "   Привет Мир     ",
			want:  "Привет Мир",
		},
		{
			name:  "Пустая строка",
			input: "",
			want:  "",
		},
		{
			name:  "Только пробелы",
			input: "       ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := norm.Normalize(tt.input)
			if result != tt.want {
				t.Errorf("Normalize(%q) = %q, expected %q", tt.input, result, tt.want)
			}
		})
	}
}
func TestRuDefaultProfile(t *testing.T) {
	norm := NewNormalizer("ru-default")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Регистр",
			input: "Привет Мир",
			want:  "привет мир",
		},
		{
			name:  "Пунктуация",
			input: "Привет, Мир!",
			want:  "привет мир",
		},
		{
			name:  "Пробелы внутри",
			input: "Привет        Мир",
			want:  "привет мир",
		},
		{
			name:  "Пробелы снаружи",
			input: "   Привет Мир     ",
			want:  "привет мир",
		},
		{
			name:  "Буква ё",
			input: "Ёжик ёжику",
			want:  "ежик ежику",
		},
		{
			name:  "Сложная строка",
			input: "    Привет, Ёжик! Как     твои дела?",
			want:  "привет ежик как твои дела",
		},
		{
			name:  "Английский язык",
			input: "  Hello World",
			want:  "hello world",
		},
		{
			name:  "Пустая строка",
			input: "",
			want:  "",
		},
		{
			name:  "Только пробелы",
			input: "       ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := norm.Normalize(tt.input)
			if result != tt.want {
				t.Errorf("Normalize(%q) = %q, expected %q", tt.input, result, tt.want)
			}
		})
	}
}
