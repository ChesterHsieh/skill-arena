package eval

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/eval"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var historyCmd = &cobra.Command{
	Use:   "history <name>",
	Short: "Show past eval run results",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistory,
}

func runHistory(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !skill.SkillExists(name) {
		return fmt.Errorf("skill %q not found at %s\n  → run 'skill-arena init %s' to create it", name, skill.SkillDir(name), name)
	}

	runs, err := eval.ListRuns(name)
	if err != nil {
		return fmt.Errorf("listing run history: %w", err)
	}

	fmt.Printf("Run history for: %s\n", name)

	if len(runs) == 0 {
		fmt.Printf("  No runs yet — run 'skill-arena eval run %s' to start.\n", name)
		return nil
	}

	// Print header
	fmt.Printf("  %-18s  %-8s  %-10s  %-8s  %s\n",
		"Timestamp", "WITH%", "WITHOUT%", "Impact", "Path")
	fmt.Printf("  %s\n", strings.Repeat("─", 72))

	for i, run := range runs {
		current := ""
		if i == 0 {
			current = "  [current]"
		}

		sign := "+"
		if run.Impact < 0 {
			sign = ""
		}

		fmt.Printf("  %-18s  %-8s  %-10s  %-8s  %s%s\n",
			run.Timestamp.Format(time.DateTime),
			fmt.Sprintf("%.0f%%", run.WithPassRate),
			fmt.Sprintf("%.0f%%", run.WithoutPassRate),
			fmt.Sprintf("%s%.0fpp", sign, run.Impact),
			run.Path,
			current,
		)
	}

	// Trend
	if len(runs) >= 2 {
		latest := runs[0].Impact
		previous := runs[1].Impact
		fmt.Printf("\n")
		if latest > previous {
			fmt.Printf("Trend: improving ↑\n")
		} else if latest < previous {
			fmt.Printf("Trend: declining ↓\n")
		} else {
			fmt.Printf("Trend: stable →\n")
		}
	}

	return nil
}
