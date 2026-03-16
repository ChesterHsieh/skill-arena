package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillsDir returns the .claude/skills path relative to the current working directory.
func SkillsDir() string {
	return filepath.Join(".claude", "skills")
}

// SkillDir returns the directory for a named skill.
func SkillDir(name string) string {
	return filepath.Join(SkillsDir(), name)
}

// EnsureSkillDir creates the skill directory and its references/ subdir.
func EnsureSkillDir(name string) error {
	dir := SkillDir(name)
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}
	return nil
}

// SkillExists returns true if the skill directory exists.
func SkillExists(name string) bool {
	_, err := os.Stat(SkillDir(name))
	return err == nil
}

// ReadSkillMeta parses the YAML frontmatter from SKILL.md in skillDir.
func ReadSkillMeta(skillDir string) (*SkillMeta, error) {
	content, err := ReadSkillContent(skillDir)
	if err != nil {
		return nil, err
	}

	frontmatter, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	meta := &SkillMeta{}
	if err := yaml.Unmarshal([]byte(frontmatter), meta); err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}
	return meta, nil
}

// ReadSkillContent returns the full content of SKILL.md in skillDir.
func ReadSkillContent(skillDir string) (string, error) {
	path := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading SKILL.md: %w", err)
	}
	return string(data), nil
}

// ReadEvalFile reads and parses evals.json from skillDir.
func ReadEvalFile(skillDir string) (*EvalFile, error) {
	path := filepath.Join(skillDir, "evals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading evals.json: %w", err)
	}

	ef := &EvalFile{}
	if err := json.Unmarshal(data, ef); err != nil {
		return nil, fmt.Errorf("parsing evals.json: %w", err)
	}
	return ef, nil
}

// WriteEvalFile serializes ef and writes it to evals.json in skillDir.
func WriteEvalFile(skillDir string, ef *EvalFile) error {
	path := filepath.Join(skillDir, "evals.json")
	data, err := json.MarshalIndent(ef, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling evals.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing evals.json: %w", err)
	}
	return nil
}

// extractFrontmatter returns the content between the first pair of --- delimiters.
func extractFrontmatter(content string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", fmt.Errorf("SKILL.md does not start with YAML frontmatter (missing opening ---)")
	}

	var fmLines []string
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(fmLines, "\n"), nil
		}
		fmLines = append(fmLines, lines[i])
	}
	return "", fmt.Errorf("SKILL.md frontmatter not closed (missing closing ---)")
}
