package eval

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/config"
	internaleval "github.com/ChesterHsieh/skill-arena/internal/eval"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run with/without skill comparison eval",
	Args:  cobra.ExactArgs(1),
	RunE:  runEval,
}

func runEval(cmd *cobra.Command, args []string) error {
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

	ef, err := skill.ReadEvalFile(skill.SkillDir(name))
	if err != nil {
		return fmt.Errorf("reading evals.json: %w", err)
	}

	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = "api.anthropic.com"
	}

	fmt.Printf("Running %d eval cases against %s (%s)\n", len(ef.Evals), cfg.DefaultModel, baseURL)
	fmt.Printf("Each case: 2 parallel requests (with skill / without skill)\n\n")

	ctx := context.Background()
	results, err := internaleval.Run(ctx, skill.SkillDir(name), cfg)
	if err != nil {
		return fmt.Errorf("eval run failed: %w", err)
	}

	// Generate and save report
	reportMD := internaleval.WriteMarkdownReport(results, name, cfg.DefaultModel)
	runDir, err := internaleval.SaveRun(name, results, reportMD)
	if err != nil {
		fmt.Printf("Warning: could not save run history: %v\n", err)
	}

	reportPath := ""
	if runDir != "" {
		reportPath = runDir + "/report.md"
	}

	internaleval.PrintInlineSummary(results, name, reportPath)

	return nil
}
