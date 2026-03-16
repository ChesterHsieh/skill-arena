package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cmeval "github.com/ChesterHsieh/skill-arena/cmd/eval"
)

var rootCmd = &cobra.Command{
	Use:   "skill-arena",
	Short: "Skill Arena — build and eval Claude Code skills",
	Long:  `skill-arena helps you scaffold, validate, and measure the impact of Claude Code skills.`,
}

// Execute runs the root command with build-time version info.
func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cmeval.EvalCmd)
}
