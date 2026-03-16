package eval

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add an eval case interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !skill.SkillExists(name) {
		return fmt.Errorf("skill %q not found at %s\n  → run 'skill-arena init %s' to create it", name, skill.SkillDir(name), name)
	}

	ef, err := skill.ReadEvalFile(skill.SkillDir(name))
	if err != nil {
		return fmt.Errorf("reading evals.json: %w", err)
	}

	// Determine available categories based on skill type
	categories := categoriesForType(ef.SkillType)

	var category string
	if err := survey.AskOne(
		&survey.Select{
			Message: "Category:",
			Options: categories,
		},
		&category,
	); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	var promptText string
	if err := survey.AskOne(
		&survey.Editor{
			Message:       "Prompt (opens editor):",
			HideDefault:   true,
			AppendDefault: true,
		},
		&promptText,
		survey.WithValidator(survey.Required),
	); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	var expectedOutput string
	if err := survey.AskOne(
		&survey.Input{
			Message: "Expected output (describe what a good response should contain):",
		},
		&expectedOutput,
		survey.WithValidator(survey.Required),
	); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	// Assertion loop
	var assertions []skill.Assertion
	assertionTypes := availableAssertionTypes(ef.SkillType)

	for {
		var addMore bool
		if err := survey.AskOne(
			&survey.Confirm{
				Message: "Add an assertion?",
				Default: true,
			},
			&addMore,
		); err != nil || !addMore {
			break
		}

		a, err := promptAssertion(assertionTypes)
		if err != nil {
			fmt.Printf("Skipping assertion: %v\n", err)
			continue
		}
		assertions = append(assertions, a)
	}

	// Compute next ID
	nextID := 1
	for _, ec := range ef.Evals {
		if ec.ID >= nextID {
			nextID = ec.ID + 1
		}
	}

	newCase := skill.EvalCase{
		ID:             nextID,
		Category:       category,
		Prompt:         promptText,
		ExpectedOutput: expectedOutput,
		Assertions:     assertions,
	}

	newEF := &skill.EvalFile{
		SkillName:    ef.SkillName,
		SkillType:    ef.SkillType,
		CodeLanguage: ef.CodeLanguage,
		LinterCmd:    ef.LinterCmd,
		Evals:        append(ef.Evals, newCase),
	}

	if err := skill.WriteEvalFile(skill.SkillDir(name), newEF); err != nil {
		return fmt.Errorf("writing evals.json: %w", err)
	}

	fmt.Printf("\n  Eval case %d added (category: %s)\n", nextID, category)
	printMissingCategories(newEF)

	return nil
}

// promptAssertion interactively collects assertion fields.
func promptAssertion(assertionTypes []string) (skill.Assertion, error) {
	var assertType string
	if err := survey.AskOne(
		&survey.Select{
			Message: "Assertion type:",
			Options: assertionTypes,
		},
		&assertType,
	); err != nil {
		return skill.Assertion{}, err
	}

	a := skill.Assertion{Type: assertType}

	switch assertType {
	case "contains", "not_contains", "step_present", "no_skipped_gate":
		var value string
		if err := survey.AskOne(
			&survey.Input{Message: "Value (string to look for):"},
			&value,
			survey.WithValidator(survey.Required),
		); err != nil {
			return skill.Assertion{}, err
		}
		a.Value = value

	case "contains_pattern":
		var pattern string
		if err := survey.AskOne(
			&survey.Input{Message: "Regex pattern:"},
			&pattern,
			survey.WithValidator(survey.Required),
		); err != nil {
			return skill.Assertion{}, err
		}
		a.Pattern = pattern

	case "step_order":
		var before, after string
		if err := survey.AskOne(&survey.Input{Message: "Step that should appear BEFORE:"}, &before, survey.WithValidator(survey.Required)); err != nil {
			return skill.Assertion{}, err
		}
		if err := survey.AskOne(&survey.Input{Message: "Step that should appear AFTER:"}, &after, survey.WithValidator(survey.Required)); err != nil {
			return skill.Assertion{}, err
		}
		a.Before = before
		a.After = after

	case "all_steps_covered":
		var stepsInput string
		if err := survey.AskOne(
			&survey.Input{Message: "Required steps (comma-separated):"},
			&stepsInput,
			survey.WithValidator(survey.Required),
		); err != nil {
			return skill.Assertion{}, err
		}
		a.Steps = splitTrimmed(stepsInput, ',')

	case "quality":
		var rubric string
		if err := survey.AskOne(
			&survey.Input{Message: "Rubric (what question should the LLM judge answer?):"},
			&rubric,
			survey.WithValidator(survey.Required),
		); err != nil {
			return skill.Assertion{}, err
		}
		a.Rubric = rubric
	}

	return a, nil
}

func categoriesForType(t skill.SkillType) []string {
	if t == skill.SkillTypeCoding {
		return []string{"core", "edge", "error_diagnosis"}
	}
	return []string{"core", "partial", "edge"}
}

func availableAssertionTypes(t skill.SkillType) []string {
	common := []string{"contains", "not_contains", "quality"}
	if t == skill.SkillTypeCoding {
		return append(common, "contains_pattern", "syntax_valid", "code_style", "compiles")
	}
	return append(common, "step_present", "step_order", "all_steps_covered", "no_skipped_gate")
}

// printMissingCategories warns if required categories are not yet covered.
func printMissingCategories(ef *skill.EvalFile) {
	required := categoriesForType(ef.SkillType)
	covered := map[string]bool{}
	for _, ec := range ef.Evals {
		covered[ec.Category] = true
	}

	var missing []string
	for _, cat := range required {
		if !covered[cat] {
			missing = append(missing, cat)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("  Still need: %d more category case(s) before eval run is unblocked (%s)\n", len(missing), joinStrings(missing, ", "))
	} else if len(ef.Evals) >= 3 {
		fmt.Printf("  All required categories covered — ready to run: skill-arena eval run %s\n", ef.SkillName)
	}
}

func splitTrimmed(s string, sep rune) []string {
	var out []string
	for _, part := range splitRune(s, sep) {
		if t := trimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func splitRune(s string, sep rune) []string {
	var parts []string
	start := 0
	for i, r := range s {
		if r == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
