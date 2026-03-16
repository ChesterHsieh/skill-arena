package skill

import "fmt"

// ScaffoldAnswers holds the answers collected from the interactive init prompts.
type ScaffoldAnswers struct {
	Name         string
	WhatItDoes   string
	TriggerWords string
	OutputFormat string
	IsCodeGen    bool
}

// GenerateSkillMD returns the content of SKILL.md based on scaffold answers.
func GenerateSkillMD(a ScaffoldAnswers) string {
	description := fmt.Sprintf("%s. Trigger when user mentions: %s.", a.WhatItDoes, a.TriggerWords)

	return fmt.Sprintf(`---
name: %s
description: >
  %s
---

# %s Skill

## When to Activate

Activate when user mentions: %s.

## Workflow

1. [Step 1 - describe first action]
2. [Step 2 - describe second action]
3. [Step 3 - describe third action]

## Output Format

%s

## Notes

- [Add any important caveats or constraints]
`, a.Name, description, titleCase(a.Name), a.TriggerWords, a.OutputFormat)
}

// GenerateEvalsTemplate returns an EvalFile with 3 template eval cases.
func GenerateEvalsTemplate(skillName string, skillType SkillType) *EvalFile {
	ef := &EvalFile{
		SkillName: skillName,
		SkillType: skillType,
	}

	switch skillType {
	case SkillTypeCoding:
		ef.CodeLanguage = "go"
		ef.Evals = []EvalCase{
			{
				ID:             1,
				Category:       "core",
				Prompt:         "[TODO: Write the most common use case prompt for this skill]",
				ExpectedOutput: "[TODO: Describe the expected code output]",
				Assertions: []Assertion{
					{Type: "syntax_valid"},
					{Type: "contains", Value: "[TODO: key identifier that should be in the output]"},
				},
			},
			{
				ID:             2,
				Category:       "edge",
				Prompt:         "[TODO: Write an unusual or complex variant prompt]",
				ExpectedOutput: "[TODO: Describe the expected output for the edge case]",
				Assertions: []Assertion{
					{Type: "syntax_valid"},
				},
			},
			{
				ID:             3,
				Category:       "error_diagnosis",
				Prompt:         "[TODO: Provide broken code or an error log for diagnosis]",
				ExpectedOutput: "[TODO: Describe the expected diagnostic output]",
				Assertions: []Assertion{
					{Type: "contains", Value: "[TODO: key fix or explanation phrase]"},
				},
			},
		}
	default: // workflow
		ef.Evals = []EvalCase{
			{
				ID:             1,
				Category:       "core",
				Prompt:         "[TODO: Write the full happy-path prompt for this workflow]",
				ExpectedOutput: "[TODO: Describe the expected full workflow output]",
				Assertions: []Assertion{
					{Type: "all_steps_covered", Steps: []string{"[step1]", "[step2]", "[step3]"}},
				},
			},
			{
				ID:             2,
				Category:       "partial",
				Prompt:         "[TODO: Write a prompt that starts the workflow mid-way]",
				ExpectedOutput: "[TODO: Describe expected output when picking up mid-workflow]",
				Assertions: []Assertion{
					{Type: "step_present", Value: "[TODO: step that must be present]"},
				},
			},
			{
				ID:             3,
				Category:       "edge",
				Prompt:         "[TODO: Write a prompt with ambiguous input or missing info]",
				ExpectedOutput: "[TODO: Describe expected graceful handling]",
				Assertions: []Assertion{
					{Type: "quality", Rubric: "Does the response handle the ambiguity gracefully?"},
				},
			},
		}
	}

	return ef
}

// titleCase converts "my-skill" or "myskill" to "My-Skill" / "Myskill".
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	result := make([]byte, len(s))
	capitalizeNext := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '_' || c == ' ' {
			result[i] = c
			capitalizeNext = true
		} else if capitalizeNext {
			if c >= 'a' && c <= 'z' {
				result[i] = c - 32
			} else {
				result[i] = c
			}
			capitalizeNext = false
		} else {
			result[i] = c
		}
	}
	return string(result)
}
