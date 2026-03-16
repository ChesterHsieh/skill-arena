package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RunSummary holds high-level stats for one historical eval run.
type RunSummary struct {
	Timestamp       time.Time
	WithPassRate    float64
	WithoutPassRate float64
	Impact          float64
	Path            string
}

// historyDir returns the path to the history directory for a skill.
func historyDir(skillName string) string {
	return filepath.Join(".skill-arena", "history", skillName)
}

// SaveRun writes the markdown report and raw results JSON for an eval run.
// It returns the directory path where the files were saved.
func SaveRun(skillName string, results []RunResult, reportMD string) (string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05")
	dir := filepath.Join(historyDir(skillName), ts)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating history directory: %w", err)
	}

	// Write markdown report
	reportPath := filepath.Join(dir, "report.md")
	if err := os.WriteFile(reportPath, []byte(reportMD), 0o644); err != nil {
		return "", fmt.Errorf("writing report.md: %w", err)
	}

	// Write raw results JSON
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling results: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "results.json"), data, 0o644); err != nil {
		return "", fmt.Errorf("writing results.json: %w", err)
	}

	return dir, nil
}

// ListRuns returns all historical run summaries for a skill, sorted newest first.
func ListRuns(skillName string) ([]RunSummary, error) {
	dir := historyDir(skillName)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading history directory: %w", err)
	}

	var summaries []RunSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		resultsPath := filepath.Join(dir, entry.Name(), "results.json")
		data, err := os.ReadFile(resultsPath)
		if err != nil {
			continue // skip incomplete runs
		}

		var results []RunResult
		if err := json.Unmarshal(data, &results); err != nil {
			continue
		}

		summary := computeSummary(results)
		summary.Path = filepath.Join(dir, entry.Name(), "report.md")

		// Parse timestamp from directory name format "2006-01-02T15-04-05"
		ts, err := time.ParseInLocation("2006-01-02T15-04-05", entry.Name(), time.Local)
		if err != nil {
			ts = time.Now()
		}
		summary.Timestamp = ts

		summaries = append(summaries, summary)
	}

	// Sort newest first
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})

	return summaries, nil
}

// computeSummary calculates aggregate pass rates from a list of run results.
func computeSummary(results []RunResult) RunSummary {
	totalWith := 0
	totalWithout := 0
	totalAssertions := 0

	for _, r := range results {
		totalWith += r.WithSkill.PassCount
		totalWithout += r.WithoutSkill.PassCount
		totalAssertions += r.WithSkill.TotalCount
	}

	if totalAssertions == 0 {
		return RunSummary{}
	}

	withRate := float64(totalWith) / float64(totalAssertions) * 100
	withoutRate := float64(totalWithout) / float64(totalAssertions) * 100

	return RunSummary{
		WithPassRate:    withRate,
		WithoutPassRate: withoutRate,
		Impact:          withRate - withoutRate,
	}
}
