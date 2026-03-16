package skill

// SkillType categorizes skills so the eval runner knows which assertion
// categories are required.
type SkillType string

const (
	SkillTypeCoding   SkillType = "coding"
	SkillTypeWorkflow SkillType = "workflow"
)

// SkillMeta holds the YAML frontmatter extracted from SKILL.md.
type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Assertion describes one assertion to evaluate against an LLM response.
type Assertion struct {
	Type    string   `json:"type"`
	Value   string   `json:"value,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Before  string   `json:"before,omitempty"`
	After   string   `json:"after,omitempty"`
	Steps   []string `json:"steps,omitempty"`
	Rubric  string   `json:"rubric,omitempty"`
}

// EvalCase is a single test case with a prompt, expected output, and assertions.
type EvalCase struct {
	ID             int         `json:"id"`
	Category       string      `json:"category"`
	Prompt         string      `json:"prompt"`
	ExpectedOutput string      `json:"expected_output"`
	Assertions     []Assertion `json:"assertions"`
}

// EvalFile is the full evals.json file for a skill.
type EvalFile struct {
	SkillName    string     `json:"skill_name"`
	SkillType    SkillType  `json:"skill_type"`
	CodeLanguage string     `json:"code_language,omitempty"`
	LinterCmd    string     `json:"linter_cmd,omitempty"`
	Evals        []EvalCase `json:"evals"`
}
