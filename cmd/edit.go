package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open SKILL.md in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !skill.SkillExists(name) {
		return fmt.Errorf("skill %q not found at %s\n  → run 'skill-arena init %s' to create it", name, skill.SkillDir(name), name)
	}

	skillMDPath := filepath.Join(skill.SkillDir(name), "SKILL.md")

	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Fallback to common editors
		for _, e := range []string{"nano", "vim", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found — set the $EDITOR environment variable")
	}

	editorCmd := exec.Command(editor, skillMDPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}
	return nil
}
