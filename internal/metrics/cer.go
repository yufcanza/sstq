package metrics

type CER struct {
	Value       float64
	Distance    int
	ManifestLen int
	H           int
	S           int
	D           int
	I           int
	Ops         []string
}

func CalculateCER(manifest, hypothesis string) CER {

	if manifest == "" && hypothesis == "" {
		return CER{
			Value:       0.0,
			Distance:    0,
			ManifestLen: 0,
			H:           0,
			S:           0,
			D:           0,
			I:           0,
			Ops:         []string{},
		}
	}
	if manifest == "" {
		hypRune := []rune(hypothesis)
		return CER{
			Value:       0.0,
			Distance:    len(hypRune),
			ManifestLen: 0,
			H:           0,
			S:           0,
			D:           0,
			I:           len(hypRune),
			Ops:         make([]string, len(hypRune)),
		}
	}
	if hypothesis == "" {
		manRune := []rune(manifest)
		return CER{
			Value:       1.0,
			Distance:    len(manRune),
			ManifestLen: len(manRune),
			H:           0,
			S:           0,
			D:           len(manRune),
			I:           0,
			Ops:         make([]string, len(manRune)),
		}
	}
	manRune := []rune(manifest)
	hypRune := []rune(hypothesis)

	manStrings := make([]string, len(manRune))
	hypStrings := make([]string, len(hypRune))

	for i, r := range manRune {
		manStrings[i] = string(r)
	}
	for i, r := range hypRune {
		hypStrings[i] = string(r)
	}

	result := Levenshtein(manStrings, hypStrings)

	hits := 0
	subs := 0
	deletions := 0
	insertions := 0

	for _, op := range result.Ops {
		switch op {
		case "M":
			hits++
		case "S":
			subs++
		case "D":
			deletions++
		case "I":
			insertions++
		}
	}
	manLen := len(manRune)
	cer := 0.0
	if manLen > 0 {
		cer = float64(result.Distance) / float64(manLen)
	}

	return CER{
		Value:       cer,
		Distance:    result.Distance,
		ManifestLen: manLen,
		H:           hits,
		S:           subs,
		D:           deletions,
		I:           insertions,
		Ops:         result.Ops,
	}

}
