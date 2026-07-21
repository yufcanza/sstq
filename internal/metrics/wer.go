package metrics

type WER struct {
	Value       float64
	Distance    int
	ManifestLen int
	H           int
	S           int
	D           int
	I           int
	Ops         []string
}

func CalculateWER(manifest, hypothesis []string) WER {
	result := Levenshtein(manifest, hypothesis)

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
	manLen := len(manifest)
	wer := 0.0
	if manLen > 0 {
		wer = float64(result.Distance) / float64(manLen)
	}

	return WER{
		Value:       wer,
		Distance:    result.Distance,
		ManifestLen: manLen,
		H:           hits,
		S:           subs,
		D:           deletions,
		I:           insertions,
		Ops:         result.Ops,
	}

}
