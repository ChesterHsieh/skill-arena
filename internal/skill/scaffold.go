package skill

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChesterHsieh/skill-arena/internal/llm"
)

// sopSystemPrompt encodes the key SKILL.md writing principles from agent-skill-sop.md.
const sopSystemPrompt = `You are an expert at writing Claude Code skill files (SKILL.md).
Follow these principles from the Agent Skill SOP:

## Description Rules (CRITICAL)
The description field is the ONLY signal the agent uses to decide whether to invoke this skill.
- Be specific and active — name the domain and concrete tasks
- List explicit trigger keywords and user intent phrases
- End with: "MUST use this skill when user mentions: <keywords>."
- Bad:  "Helps users process stream data"
- Good: "Apache Flink streaming expert. Helps write DataStream API, Table API code, design
         Flink Job architectures, diagnose checkpoint/backpressure/watermark issues.
         MUST use this skill when user mentions: Flink, stream processing, Kafka source,
         watermark, backpressure, state backend, exactly-once, CEP."

## Required Body Sections (in order)
1. ## When to Activate
   - List specific trigger scenarios as bullet points

2. ## Workflow
   Choose A/B/C paths based on user intent. Each path has numbered steps.
   Example paths: A. Code Generation  B. Architecture Design  C. Error Diagnosis
   Explain WHY each step matters — don't just list actions.

3. ## Output Format
   ALWAYS specify the exact response structure with concrete sub-sections.
   Example:
     ### Code
     [executable code with full imports]
     ### Explanation
     [architecture decision rationale, max 200 words]
     ### Caveats
     [performance issues, compatibility notes]

4. ## Notes
   Version compatibility, language differences, common pitfalls.

## Constraints
- Body must be ≤ 500 lines
- Use "ALWAYS"/"NEVER" sparingly — prefer reasoning over commands
- Write in English`

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

// GenerateSkillMDWithLLM calls the LLM using the SOP as a system prompt to produce
// a well-structured SKILL.md from the raw scaffold answers. Falls back to
// GenerateSkillMD if the LLM call fails.
func GenerateSkillMDWithLLM(ctx context.Context, a ScaffoldAnswers, client *llm.Client) (string, bool, error) {
	skillTypeLabel := "workflow"
	if a.IsCodeGen {
		skillTypeLabel = "coding (code-generation)"
	}

	userPrompt := fmt.Sprintf(`Generate a complete SKILL.md for a new Claude Code skill using the following answers collected from the skill author.
Transform these raw answers into a well-structured, production-quality SKILL.md following the SOP principles.

Skill name: %s
Skill type: %s

Raw answers from author:
  What the AI should do: %s
  Trigger keywords / user intent: %s
  Expected output format: %s

Requirements:
- Return ONLY the raw SKILL.md content — no explanation, no markdown code fences
- Start directly with the --- frontmatter block
- Write a description that is specific, active, and includes explicit trigger keywords
- Expand the workflow section with realistic, concrete steps (not placeholders)
- Make the Output Format section match exactly what was described
- Keep body ≤ 500 lines`,
		a.Name, skillTypeLabel, a.WhatItDoes, a.TriggerWords, a.OutputFormat,
	)

	resp, err := client.Complete(ctx, sopSystemPrompt, userPrompt)
	if err != nil {
		return GenerateSkillMD(a), false, err
	}

	content := strings.TrimSpace(resp.Content)

	// Strip accidental ```markdown fences if the model added them
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		// drop first line (```markdown or ```) and last line (```)
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	return content, true, nil
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
