package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/config"
	"github.com/ChesterHsieh/skill-arena/internal/llm"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var generateCmd = &cobra.Command{
	Use:   "generate <name>",
	Short: "Use LLM to suggest eval cases from SKILL.md",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerate,
}

type suggestedCase struct {
	Category       string           `json:"category"`
	Prompt         string           `json:"prompt"`
	ExpectedOutput string           `json:"expected_output"`
	Assertions     []skill.Assertion `json:"assertions,omitempty"`
}

func runGenerate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !skill.SkillExists(name) {
		return fmt.Errorf("skill %q not found at %s\n  → run 'skill-arena init %s' to create it", name, skill.SkillDir(name), name)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured\n  → run 'skill-arena config' to set your API key")
	}

	fmt.Printf("Reading SKILL.md...\n")

	skillContent, err := skill.ReadSkillContent(skill.SkillDir(name))
	if err != nil {
		return fmt.Errorf("reading skill content: %w", err)
	}

	ef, err := skill.ReadEvalFile(skill.SkillDir(name))
	if err != nil {
		return fmt.Errorf("reading evals.json: %w", err)
	}

	client := llm.NewClient(cfg)
	fmt.Printf("Calling %s to suggest eval cases...\n\n", cfg.DefaultModel)

	suggestions, err := generateEvalSuggestions(context.Background(), client, skillContent, ef.SkillType)
	if err != nil {
		return fmt.Errorf("generating suggestions: %w", err)
	}

	if len(suggestions) == 0 {
		fmt.Printf("No suggestions returned. Try refining your SKILL.md.\n")
		return nil
	}

	// Display suggestions
	fmt.Printf("Suggested eval cases:\n")
	options := make([]string, len(suggestions))
	for i, s := range suggestions {
		label := fmt.Sprintf("[%d] %-16s — %q", i+1, s.Category, truncate(s.Prompt, 60))
		options[i] = label
		fmt.Printf("  %s\n", label)
	}

	// Multi-select
	var selectedLabels []string
	if err := survey.AskOne(
		&survey.MultiSelect{
			Message: "Which cases to add? (space to select, enter to confirm)",
			Options: options,
		},
		&selectedLabels,
	); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if len(selectedLabels) == 0 {
		fmt.Printf("No cases selected.\n")
		return nil
	}

	// Build a set of selected indices
	selected := map[int]bool{}
	for _, label := range selectedLabels {
		for i, opt := range options {
			if opt == label {
				selected[i] = true
			}
		}
	}

	// Compute next ID
	nextID := 1
	for _, ec := range ef.Evals {
		if ec.ID >= nextID {
			nextID = ec.ID + 1
		}
	}

	var added []skill.EvalCase
	for i, s := range suggestions {
		if !selected[i] {
			continue
		}
		assertions := s.Assertions
		if len(assertions) == 0 {
			assertions = defaultAssertions(ef.SkillType)
		}
		ec := skill.EvalCase{
			ID:             nextID,
			Category:       s.Category,
			Prompt:         s.Prompt,
			ExpectedOutput: s.ExpectedOutput,
			Assertions:     assertions,
		}
		ef.Evals = append(ef.Evals, ec)
		added = append(added, ec)
		nextID++
	}

	if err := skill.WriteEvalFile(skill.SkillDir(name), ef); err != nil {
		return fmt.Errorf("writing evals.json: %w", err)
	}

	fmt.Printf("\n  %d case(s) added to evals.json\n", len(added))
	printMissingCategories(ef)

	return nil
}

// generateEvalSuggestions calls the LLM to produce eval case suggestions.
func generateEvalSuggestions(ctx context.Context, client *llm.Client, skillContent string, skillType skill.SkillType) ([]suggestedCase, error) {
	systemPrompt := `You are an expert at writing LLM eval test cases.
You produce JSON arrays of eval cases that stress-test AI skill descriptions.
Each case should test different aspects: core functionality, edge cases, error handling.
Always respond with a valid JSON array and nothing else (no markdown, no explanation).`

	requiredCats := "core, edge, error_diagnosis"
	assertionExample := `[
        {"type": "syntax_valid"},
        {"type": "contains", "value": "specific_function_or_keyword"},
        {"type": "quality", "rubric": "Does the response correctly implement X with proper error handling?"}
      ]`
	if skillType == skill.SkillTypeWorkflow {
		requiredCats = "core, partial, edge"
		assertionExample = `[
        {"type": "all_steps_covered", "steps": ["step A", "step B", "step C"]},
        {"type": "step_order", "before": "step A", "after": "step B"},
        {"type": "quality", "rubric": "Does the response follow the prescribed workflow order?"}
      ]`
	}

	userPrompt := fmt.Sprintf(`Given the following SKILL.md content, generate 4-6 diverse eval cases that stress-test its claimed capabilities.

Skill type: %s
Required categories (include at least one of each): %s

For EACH case, include specific assertions derived from the expected_output — not generic placeholders.
- "contains": use a concrete keyword, function name, or phrase that MUST appear in a correct answer
- "quality" rubric: describe exactly what makes a good response for THIS specific prompt
- "all_steps_covered": list the actual step names from the workflow, not generic labels
- "syntax_valid": include for any case expecting code output

Return a JSON array in this exact format:
[
  {
    "category": "core",
    "prompt": "...",
    "expected_output": "...",
    "assertions": %s
  }
]

SKILL.md:
%s`, skillType, requiredCats, assertionExample, skillContent)

	resp, err := client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON — handle if wrapped in ```json ... ``` blocks
	jsonStr := extractJSON(resp.Content)

	var suggestions []suggestedCase
	if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
		return nil, fmt.Errorf("parsing LLM response as JSON: %w\n\nRaw response:\n%s", err, resp.Content)
	}

	return suggestions, nil
}

// extractJSON strips ```json ... ``` fences if present, then extracts the
// outermost [...] array from the result. Handles all common model output formats.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Strip code fences (```json, ```JSON, ``` — any variant, optional newline after)
	re := regexp.MustCompile("(?si)```(?:json)?\r?\n?(.*?)```")
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}

	// Extract the outermost [...] array
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return strings.TrimSpace(s[start : end+1])
	}

	return s
}

// defaultAssertions returns sensible default assertions for a skill type.
func defaultAssertions(t skill.SkillType) []skill.Assertion {
	if t == skill.SkillTypeCoding {
		return []skill.Assertion{
			{Type: "syntax_valid"},
			{Type: "contains", Value: ""},
		}
	}
	return []skill.Assertion{
		{Type: "quality", Rubric: "Does the response adequately address the prompt?"},
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
