package structs

type Version struct {
	Major int
	Minor int
	Patch int
}

type Result struct {
	Name        string
	Version     string
	Description string
}

type Variable struct {
	Description  string `yaml:"Description"`
	ExampleValue string `yaml:"ExampleValue"`
	DefaultValue string `yaml:"DefaultValue"`
}
type Snippet struct {
	Name        string `yaml:"Name"`
	Version     string `yaml:"Version"`
	Description string `yaml:"Description"`
	Variables   map[string]*Variable
	Tags        []string `yaml:"Tags"`
	SnippetFile string   `yaml:"SnippetFile"`
	SnippetText string   `yaml:"SnippetText"`
}
type Config struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
	Host      string `yaml:"host"`
	Port      string `yaml:"port"`
	OutputDir string `yaml:"outputdir"`
}
type ConfigLocal struct {
	SQLFile   string `yaml:"sqlFile"`
	OutputDir string `yaml:"outputdir"`
}
