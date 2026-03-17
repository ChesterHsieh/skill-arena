package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/ChesterHsieh/skill-arena/internal/config"
	"github.com/ChesterHsieh/skill-arena/internal/llm"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Scaffold a new skill interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	if skill.SkillExists(name) {
		return fmt.Errorf("skill %q already exists at %s\n  → use 'skill-arena edit %s' to modify it", name, skill.SkillDir(name), name)
	}

	answers := skill.ScaffoldAnswers{Name: name}

	questions := []*survey.Question{
		{
			Name:     "WhatItDoes",
			Prompt:   &survey.Input{Message: "What should the AI do with this skill?"},
			Validate: survey.Required,
		},
		{
			Name:     "TriggerWords",
			Prompt:   &survey.Input{Message: "When should it trigger? (list keywords / user intent)"},
			Validate: survey.Required,
		},
		{
			Name:     "OutputFormat",
			Prompt:   &survey.Input{Message: "What is the expected output format?"},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(questions, &answers); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	var isCodeGen bool
	if err := survey.AskOne(
		&survey.Confirm{
			Message: "Is this a code-generation skill?",
			Default: false,
		},
		&isCodeGen,
	); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}
	answers.IsCodeGen = isCodeGen

	var skillType skill.SkillType
	if isCodeGen {
		skillType = skill.SkillTypeCoding
	} else {
		skillType = skill.SkillTypeWorkflow
	}

	// Create directory structure
	if err := skill.EnsureSkillDir(name); err != nil {
		return err
	}

	// Generate SKILL.md — use LLM if API key is configured, else fall back to template
	var skillMD string
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg.APIKey != "" {
		fmt.Printf("\n  Generating structured SKILL.md using SOP guidelines...\n")
		client := llm.NewClient(cfg)
		var usedLLM bool
		var genErr error
		skillMD, usedLLM, genErr = skill.GenerateSkillMDWithLLM(context.Background(), answers, client)
		if genErr != nil {
			fmt.Printf("  ⚠ LLM generation failed (%v), using template instead\n", genErr)
		} else if usedLLM {
			fmt.Printf("  ✓ SKILL.md structured by LLM (model: %s)\n", cfg.DefaultModel)
		}
	} else {
		skillMD = skill.GenerateSkillMD(answers)
	}

	skillMDPath := filepath.Join(skill.SkillDir(name), "SKILL.md")
	if err := os.WriteFile(skillMDPath, []byte(skillMD), 0o644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	// Write evals.json
	evalFile := skill.GenerateEvalsTemplate(name, skillType)
	evalsData, err := json.MarshalIndent(evalFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling evals.json: %w", err)
	}
	evalsPath := filepath.Join(skill.SkillDir(name), "evals.json")
	if err := os.WriteFile(evalsPath, evalsData, 0o644); err != nil {
		return fmt.Errorf("writing evals.json: %w", err)
	}

	fmt.Printf("\n")
	fmt.Printf("  Created %s\n", skillMDPath)
	fmt.Printf("  Created %s  (template with 3 empty cases)\n", evalsPath)
	fmt.Printf("  Created %s\n", filepath.Join(skill.SkillDir(name), "references/"))
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Edit your skill:      skill-arena edit %s\n", name)
	fmt.Printf("  2. Add eval cases:       skill-arena eval add %s\n", name)
	fmt.Printf("     (or generate them:    skill-arena eval generate %s)\n", name)
	fmt.Printf("  3. Run evals:            skill-arena eval run %s\n", name)

	return nil
}
