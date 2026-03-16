package eval

import "github.com/spf13/cobra"

// EvalCmd is the parent command for all eval subcommands.
var EvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Manage and run skill evaluations",
}

func init() {
	EvalCmd.AddCommand(addCmd)
	EvalCmd.AddCommand(generateCmd)
	EvalCmd.AddCommand(runCmd)
	EvalCmd.AddCommand(historyCmd)
}
