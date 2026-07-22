package corpus

type Manifest struct {
	ID   string   `json:"id"`
	Text string   `json:"text"`
	Tags []string `json:"tags,omitempty"`
}
type Hypothesis struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
