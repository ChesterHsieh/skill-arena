package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Set API endpoint, API key, and default model",
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("Configuring skill-arena  (stored at %s)\n\n", config.ConfigPath())

	var answers struct {
		APIBaseURL   string
		APIKey       string
		DefaultModel string
		LinterPath   string
	}

	questions := []*survey.Question{
		{
			Name: "APIBaseURL",
			Prompt: &survey.Input{
				Message: "API base URL",
				Default: cfg.APIBaseURL,
			},
			Validate: survey.Required,
		},
		{
			Name: "APIKey",
			Prompt: &survey.Password{
				Message: "API key",
			},
		},
		{
			Name: "DefaultModel",
			Prompt: &survey.Input{
				Message: "Default model",
				Default: cfg.DefaultModel,
			},
			Validate: survey.Required,
		},
		{
			Name: "LinterPath",
			Prompt: &survey.Input{
				Message: "Linter path (optional, e.g. 'golangci-lint', 'ruff')",
				Default: cfg.LinterPath,
			},
		},
	}

	if err := survey.Ask(questions, &answers); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	// Only update API key if a new one was entered
	newCfg := &config.Config{
		APIBaseURL:   answers.APIBaseURL,
		DefaultModel: answers.DefaultModel,
		LinterPath:   answers.LinterPath,
	}
	if answers.APIKey != "" {
		newCfg.APIKey = answers.APIKey
	} else {
		newCfg.APIKey = cfg.APIKey
	}

	if err := config.Save(newCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", config.ConfigPath())
	return nil
}
