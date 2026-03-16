package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var validateCmd = &cobra.Command{
	Use:   "validate <name>",
	Short: "Check SKILL.md against SOP rules",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !skill.SkillExists(name) {
		return fmt.Errorf("skill %q not found at %s\n  → run 'skill-arena init %s' to create it", name, skill.SkillDir(name), name)
	}

	results, err := skill.Validate(skill.SkillDir(name))
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	errorCount := 0
	warnCount := 0

	for _, r := range results {
		switch r.Severity {
		case skill.SeverityOK:
			fmt.Printf("  %s %s: %s\n", checkMark(), r.Field, r.Message)
		case skill.SeverityWarning:
			fmt.Printf("  %s %s: %s\n", warnMark(), r.Field, r.Message)
			warnCount++
		case skill.SeverityError:
			fmt.Printf("  %s %s: %s\n", errorMark(), r.Field, r.Message)
			errorCount++
		}
	}

	fmt.Printf("\n")
	if errorCount == 0 && warnCount == 0 {
		fmt.Printf("All checks passed.\n")
	} else {
		fmt.Printf("%d warning(s), %d error(s)\n", warnCount, errorCount)
	}

	if errorCount > 0 {
		os.Exit(1)
	}
	return nil
}

func checkMark() string { return "✓" }
func warnMark() string  { return "⚠" }
func errorMark() string { return "✗" }
