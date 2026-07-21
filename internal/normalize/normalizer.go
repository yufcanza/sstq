package normalize

type Normalizer struct {
	profile *Profile
}

func NewNormalizer(profileName string) *Normalizer {
	return &Normalizer{
		profile: GetProfile(profileName),
	}
}

func (n Normalizer) Normalize(text string) string {
	result := text
	for _, transform := range n.profile.Transforms {
		result = transform.Apply(result)
	}
	return result
}
