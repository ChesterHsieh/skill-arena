package eval

import (
	"fmt"
	"strings"
	"time"
)

// PrintInlineSummary prints a terminal-friendly summary of eval results.
func PrintInlineSummary(results []RunResult, skillName string, reportPath string) {
	fmt.Printf("\n")
	for i, r := range results {
		prompt := r.Prompt
		if len(prompt) > 50 {
			prompt = prompt[:47] + "..."
		}
		fmt.Printf("Case %d/%d [%s]: %q\n", i+1, len(results), r.Category, prompt)

		withTokens := r.WithSkill.InputTokens + r.WithSkill.OutputTokens
		withoutTokens := r.WithoutSkill.InputTokens + r.WithoutSkill.OutputTokens
		fmt.Printf("  WITH    → %d tokens  [assertions: %d/%d passed]\n",
			withTokens, r.WithSkill.PassCount, r.WithSkill.TotalCount)
		fmt.Printf("  WITHOUT → %d tokens  [assertions: %d/%d passed]\n",
			withoutTokens, r.WithoutSkill.PassCount, r.WithoutSkill.TotalCount)

		// Print truncated diff inline (max 30 lines)
		if r.Diff != "" {
			diffLines := strings.Split(r.Diff, "\n")
			if len(diffLines) > 30 {
				diffLines = append(diffLines[:30], "  ... (truncated, see full report)")
			}
			fmt.Printf("  Diff:\n")
			for _, line := range diffLines {
				if line != "" {
					fmt.Printf("    %s\n", line)
				}
			}
		}
		fmt.Printf("\n")
	}

	// Compute summary statistics
	totalWith := 0
	totalWithout := 0
	totalAssertions := 0
	totalWithTokens := 0
	totalWithoutTokens := 0

	for _, r := range results {
		totalWith += r.WithSkill.PassCount
		totalWithout += r.WithoutSkill.PassCount
		totalAssertions += r.WithSkill.TotalCount
		totalWithTokens += r.WithSkill.InputTokens + r.WithSkill.OutputTokens
		totalWithoutTokens += r.WithoutSkill.InputTokens + r.WithoutSkill.OutputTokens
	}

	sep := strings.Repeat("─", 44)
	fmt.Printf("%s\n", sep)

	withPct := 0.0
	withoutPct := 0.0
	if totalAssertions > 0 {
		withPct = float64(totalWith) / float64(totalAssertions) * 100
		withoutPct = float64(totalWithout) / float64(totalAssertions) * 100
	}
	impact := withPct - withoutPct

	avgTokenDelta := 0
	if len(results) > 0 {
		avgTokenDelta = (totalWithTokens - totalWithoutTokens) / len(results)
	}

	fmt.Printf("Summary  WITH:    %d/%d assertions passed (%.0f%%)\n", totalWith, totalAssertions, withPct)
	fmt.Printf("         WITHOUT: %d/%d assertions passed (%.0f%%)\n", totalWithout, totalAssertions, withoutPct)

	sign := "+"
	if impact < 0 {
		sign = ""
	}
	fmt.Printf("         Skill impact: %s%.0fpp assertion improvement\n", sign, impact)

	tokenSign := "+"
	if avgTokenDelta < 0 {
		tokenSign = ""
	}
	fmt.Printf("         Avg token delta: %s%d tokens with skill (cost tradeoff)\n", tokenSign, avgTokenDelta)
	fmt.Printf("%s\n\n", sep)

	if reportPath != "" {
		fmt.Printf("Full report: %s\n", reportPath)
		fmt.Printf("Run 'skill-arena eval history %s' to browse history\n", skillName)
	}
}

// WriteMarkdownReport generates a full markdown report of eval results.
func WriteMarkdownReport(results []RunResult, skillName string, model string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Eval Report: %s\n\n", skillName))
	sb.WriteString(fmt.Sprintf("**Timestamp:** %s  \n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Model:** %s  \n", model))
	sb.WriteString(fmt.Sprintf("**Cases:** %d  \n\n", len(results)))
	sb.WriteString("---\n\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("## Case %d — [%s]\n\n", r.EvalID, r.Category))
		sb.WriteString("**Prompt:**\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", r.Prompt))

		// WITH section
		withTokens := r.WithSkill.InputTokens + r.WithSkill.OutputTokens
		sb.WriteString(fmt.Sprintf("### WITH Skill (%d tokens, %d/%d assertions)\n\n",
			withTokens, r.WithSkill.PassCount, r.WithSkill.TotalCount))
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", r.WithSkill.Response))

		if len(r.WithSkill.Assertions) > 0 {
			sb.WriteString("| Assertion | Passed | Message |\n")
			sb.WriteString("|-----------|--------|---------|\n")
			for _, a := range r.WithSkill.Assertions {
				icon := "✓"
				if !a.Passed {
					icon = "✗"
				}
				msg := strings.ReplaceAll(a.Message, "|", "\\|")
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", a.Type, icon, msg))
			}
			sb.WriteString("\n")
		}

		// WITHOUT section
		withoutTokens := r.WithoutSkill.InputTokens + r.WithoutSkill.OutputTokens
		sb.WriteString(fmt.Sprintf("### WITHOUT Skill (%d tokens, %d/%d assertions)\n\n",
			withoutTokens, r.WithoutSkill.PassCount, r.WithoutSkill.TotalCount))
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", r.WithoutSkill.Response))

		if len(r.WithoutSkill.Assertions) > 0 {
			sb.WriteString("| Assertion | Passed | Message |\n")
			sb.WriteString("|-----------|--------|---------|\n")
			for _, a := range r.WithoutSkill.Assertions {
				icon := "✓"
				if !a.Passed {
					icon = "✗"
				}
				msg := strings.ReplaceAll(a.Message, "|", "\\|")
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", a.Type, icon, msg))
			}
			sb.WriteString("\n")
		}

		// Diff section
		if r.Diff != "" {
			sb.WriteString("### Diff (without → with)\n\n")
			sb.WriteString("```diff\n")
			sb.WriteString(r.Diff)
			sb.WriteString("\n```\n\n")
		}

		sb.WriteString("---\n\n")
	}

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Case | Category | WITH | WITHOUT | Impact |\n")
	sb.WriteString("|------|----------|------|---------|--------|\n")

	totalWith := 0
	totalWithout := 0
	totalAssertions := 0

	for _, r := range results {
		totalWith += r.WithSkill.PassCount
		totalWithout += r.WithoutSkill.PassCount
		totalAssertions += r.WithSkill.TotalCount

		withPct := 0.0
		withoutPct := 0.0
		if r.WithSkill.TotalCount > 0 {
			withPct = float64(r.WithSkill.PassCount) / float64(r.WithSkill.TotalCount) * 100
			withoutPct = float64(r.WithoutSkill.PassCount) / float64(r.WithoutSkill.TotalCount) * 100
		}
		impact := withPct - withoutPct
		sign := "+"
		if impact < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %.0f%% | %.0f%% | %s%.0fpp |\n",
			r.EvalID, r.Category, withPct, withoutPct, sign, impact))
	}

	sb.WriteString("\n")

	overallWith := 0.0
	overallWithout := 0.0
	if totalAssertions > 0 {
		overallWith = float64(totalWith) / float64(totalAssertions) * 100
		overallWithout = float64(totalWithout) / float64(totalAssertions) * 100
	}
	overallImpact := overallWith - overallWithout
	sign := "+"
	if overallImpact < 0 {
		sign = ""
	}

	sb.WriteString(fmt.Sprintf("**Overall:** WITH %.0f%% | WITHOUT %.0f%% | Impact %s%.0fpp\n", overallWith, overallWithout, sign, overallImpact))

	return sb.String()
}
