package eiffel

const BasicTemplateName = "ebt"

type BasicTemplate struct {
	Name          string
	Version       string
	Authors       []string
	License       string
	Description   string
	Format        string
	Example       string
	Preprocessors []BasicPreprocessor
	Rules         map[string]BasicRule
	Variants      map[string]BasicVariant
}

type BasicRule struct {
	Name        string
	Explanation string
	Description string
	Type        string
	Value       any
	Optional    bool
}

type BasicVariant struct {
	Name    string
	Format  string
	Example string
	Rules   []string // Rules contains rule names, rule objects should be contained in the template
}

type BasicPreprocessor func([]byte) error
