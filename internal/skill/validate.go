package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Severity indicates how serious a validation finding is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityOK      Severity = "ok"
)

// ValidationResult holds a single validation finding.
type ValidationResult struct {
	Severity Severity
	Field    string
	Message  string
}

// Validate runs all SOP compliance checks on the skill at skillDir and returns
// a slice of results. An error is returned only for I/O failures.
func Validate(skillDir string) ([]ValidationResult, error) {
	var results []ValidationResult

	content, err := ReadSkillContent(skillDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read SKILL.md: %w", err)
	}

	meta, metaErr := ReadSkillMeta(skillDir)

	// Rule: frontmatter name present
	if metaErr != nil || meta == nil || meta.Name == "" {
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Field:    "name",
			Message:  "frontmatter 'name' is missing — add 'name: <skill-name>' to the YAML header",
		})
	} else {
		results = append(results, ValidationResult{
			Severity: SeverityOK,
			Field:    "name",
			Message:  fmt.Sprintf("present (%q)", meta.Name),
		})
	}

	// Rule: frontmatter description present and >= 50 chars
	if metaErr != nil || meta == nil || len(strings.TrimSpace(meta.Description)) < 50 {
		descLen := 0
		if meta != nil {
			descLen = len(strings.TrimSpace(meta.Description))
		}
		if descLen == 0 {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Field:    "description",
				Message:  "frontmatter 'description' is missing — add a description of at least 50 characters",
			})
		} else {
			results = append(results, ValidationResult{
				Severity: SeverityError,
				Field:    "description",
				Message:  fmt.Sprintf("%d chars — description must be at least 50 characters (currently too short)", descLen),
			})
		}
	} else {
		results = append(results, ValidationResult{
			Severity: SeverityOK,
			Field:    "description",
			Message:  fmt.Sprintf("%d chars (min 50)", len(strings.TrimSpace(meta.Description))),
		})
	}

	// Rule: description contains at least one trigger keyword
	if meta != nil && len(strings.TrimSpace(meta.Description)) >= 50 {
		triggerPhrases := extractTriggerPhrases(meta.Description)
		if len(triggerPhrases) == 0 {
			results = append(results, ValidationResult{
				Severity: SeverityWarning,
				Field:    "description",
				Message:  "description does not appear to contain trigger keywords — add 'Trigger when user mentions: ...' or list key phrases",
			})
		} else {
			results = append(results, ValidationResult{
				Severity: SeverityOK,
				Field:    "description",
				Message:  fmt.Sprintf("contains trigger keywords (%s)", strings.Join(triggerPhrases[:min(3, len(triggerPhrases))], ", ")),
			})
		}
	}

	// Rule: body line count — warning at 400, error at 500
	lines := strings.Split(content, "\n")
	bodyLines := countBodyLines(lines)
	switch {
	case bodyLines > 500:
		results = append(results, ValidationResult{
			Severity: SeverityError,
			Field:    "body",
			Message:  fmt.Sprintf("%d lines (error threshold: 500) — split large sections into references/ files", bodyLines),
		})
	case bodyLines > 400:
		results = append(results, ValidationResult{
			Severity: SeverityWarning,
			Field:    "body",
			Message:  fmt.Sprintf("%d lines (warning threshold: 400) — consider moving detail to references/", bodyLines),
		})
	default:
		results = append(results, ValidationResult{
			Severity: SeverityOK,
			Field:    "body",
			Message:  fmt.Sprintf("%d lines (max 500)", bodyLines),
		})
	}

	// Rule: has ## Output Format section
	if !strings.Contains(content, "## Output Format") {
		results = append(results, ValidationResult{
			Severity: SeverityWarning,
			Field:    "body",
			Message:  "missing '## Output Format' section — add a section describing the expected response structure",
		})
	} else {
		results = append(results, ValidationResult{
			Severity: SeverityOK,
			Field:    "body",
			Message:  "## Output Format section present",
		})
	}

	// Rule: if body references external docs, references/ should have at least one file
	refsDir := filepath.Join(skillDir, "references")
	if strings.Contains(strings.ToLower(content), "references/") {
		entries, _ := os.ReadDir(refsDir)
		if len(entries) == 0 {
			results = append(results, ValidationResult{
				Severity: SeverityWarning,
				Field:    "references",
				Message:  "SKILL.md references 'references/' but the directory is empty — add referenced files",
			})
		} else {
			results = append(results, ValidationResult{
				Severity: SeverityOK,
				Field:    "references",
				Message:  fmt.Sprintf("%d file(s) in references/", len(entries)),
			})
		}
	}

	return results, nil
}

// extractTriggerPhrases finds keywords after common trigger phrases in the description.
func extractTriggerPhrases(description string) []string {
	lower := strings.ToLower(description)
	var phrases []string

	markers := []string{
		"trigger when user mentions:",
		"trigger when",
		"activate when",
		"use this skill when",
		"when user mentions",
	}

	for _, marker := range markers {
		idx := strings.Index(lower, marker)
		if idx == -1 {
			continue
		}
		after := description[idx+len(marker):]
		// Split on commas, semicolons, or end of sentence
		parts := strings.FieldsFunc(after, func(r rune) bool {
			return r == ',' || r == ';' || r == '.' || r == '\n'
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" && len(p) < 60 {
				phrases = append(phrases, p)
			}
			if len(phrases) >= 5 {
				break
			}
		}
		break
	}
	return phrases
}

// countBodyLines counts lines that are part of the body (after frontmatter).
func countBodyLines(lines []string) int {
	inFrontmatter := false
	pastFrontmatter := false
	count := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !pastFrontmatter {
				if !inFrontmatter {
					inFrontmatter = true
				} else {
					inFrontmatter = false
					pastFrontmatter = true
				}
				continue
			}
		}
		if pastFrontmatter {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
