package normalize

type Profile struct {
	Name       string
	Transforms []Transform
}

func NewStrictProfile() *Profile {
	return &Profile{
		Name: "strict",
		Transforms: []Transform{
			NFCTransform{},
			CollapseSpacesTransform{},
			TrimSpaceTransform{},
		},
	}
}

func NewRuDefaultProfile() *Profile {
	return &Profile{
		Name: "ru-default",
		Transforms: []Transform{
			NFCTransform{},
			LowerCaseTransform{},
			ReplaceTransform{},
			PunctuationToSpaceTransform{},
			CollapseSpacesTransform{},
			TrimSpaceTransform{},
		},
	}
}

func GetProfile(name string) *Profile {
	switch name {
	case "strict":
		return NewStrictProfile()
	case "ru-default":
		return NewRuDefaultProfile()
	default:
		return NewStrictProfile()
	}
}
