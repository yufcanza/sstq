package metrics

type AlignmentItem struct {
	Type       string `json:"type"`
	Manifest   string `json:"reference"`
	Hypothesis string `json:"hypothesis"`
}

type Alignment struct {
	Items []AlignmentItem
}

func CreateAlignment(manifest, hypothesis []string) Alignment {
	result := Levenshtein(manifest, hypothesis)

	items := make([]AlignmentItem, 0, len(result.Ops))
	i, j := 0, 0
	for _, op := range result.Ops {
		switch op {
		case "M":
			items = append(items, AlignmentItem{
				Type:       "equal",
				Manifest:   manifest[i],
				Hypothesis: hypothesis[j],
			})
			i++
			j++
		case "S":
			items = append(items, AlignmentItem{
				Type:       "substitute",
				Manifest:   manifest[i],
				Hypothesis: hypothesis[j],
			})
			i++
			j++
		case "D":
			items = append(items, AlignmentItem{
				Type:       "delete",
				Manifest:   manifest[i],
				Hypothesis: "",
			})
			i++
		case "I":
			items = append(items, AlignmentItem{
				Type:       "insert",
				Manifest:   "",
				Hypothesis: hypothesis[j],
			})
			j++
		}

	}
	return Alignment{Items: items}
}
