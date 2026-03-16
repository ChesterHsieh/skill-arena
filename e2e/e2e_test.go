//go:build e2e

package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// binaryOnce ensures the binary is only built once across all tests.
var (
	binaryOnce sync.Once
	binaryPath string
	binaryErr  error
)

// buildBinary builds the skill-arena binary to a temp directory and returns
// its path. Results are cached so the binary is only built once per test run.
func buildBinary(t *testing.T) string {
	t.Helper()
	binaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "skill-arena-bin-*")
		if err != nil {
			binaryErr = err
			return
		}
		out := filepath.Join(dir, "skill-arena")
		// Resolve project root relative to this test file's location.
		// __file__ is not available in Go, so we find it via the module.
		projectRoot, err := findProjectRoot()
		if err != nil {
			binaryErr = err
			return
		}
		cmd := exec.Command("go", "build", "-o", out, ".")
		cmd.Dir = projectRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			binaryErr = &buildError{cause: err, output: string(output)}
			return
		}
		binaryPath = out
	})

	if binaryErr != nil {
		t.Fatalf("failed to build skill-arena binary: %v", binaryErr)
	}
	return binaryPath
}

// buildError wraps a build failure with compiler output for better diagnostics.
type buildError struct {
	cause  error
	output string
}

func (e *buildError) Error() string {
	return e.cause.Error() + "\n" + e.output
}

// findProjectRoot locates the module root by searching upward from this file's
// directory for a go.mod containing the expected module name.
func findProjectRoot() (string, error) {
	// os.Getwd returns the working directory at test start — for `go test ./e2e/`
	// that is the e2e/ directory itself.
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		gomod := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(gomod)
		if err == nil && strings.Contains(string(data), "github.com/ChesterHsieh/skill-arena") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", &missingModuleError{start: cwd}
}

type missingModuleError struct{ start string }

func (e *missingModuleError) Error() string {
	return "could not find go.mod for github.com/ChesterHsieh/skill-arena starting from " + e.start
}

// setupSkillDir creates a temp working directory with the given example skill
// copied into .claude/skills/<name>/. It returns the path to the temp dir.
func setupSkillDir(t *testing.T, exampleName string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "skill-arena-wd-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Locate the example in the project tree.
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("finding project root: %v", err)
	}
	exampleSrc := filepath.Join(projectRoot, "examples", exampleName)

	skillsDest := filepath.Join(tmpDir, ".claude", "skills", exampleName)
	if err := copyDir(exampleSrc, skillsDest); err != nil {
		t.Fatalf("copying example skill %q: %v", exampleName, err)
	}

	return tmpDir
}

// copyDir recursively copies src into dst, creating dst and all parents.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// runCmd runs the skill-arena binary with the given args from workDir,
// injecting env into the subprocess environment. It returns stdout, stderr,
// and the exit code.
func runCmd(t *testing.T, binary string, workDir string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = workDir
	// Start with the host environment so PATH and shared libs are available,
	// then override / append caller-supplied variables.
	cmd.Env = append(os.Environ(), env...)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start — treat as exit 1.
			exitCode = 1
		}
	}
	return stdout, stderr, exitCode
}

// ---- Unit-level E2E tests (no API key required) ----

// TestValidateFlink runs `skill-arena validate flink-skill` and asserts that
// the example SKILL.md passes all required SOP checks with no errors.
func TestValidateFlink(t *testing.T) {
	binary := buildBinary(t)
	workDir := setupSkillDir(t, "flink-skill")

	stdout, stderr, exitCode := runCmd(t, binary, workDir, nil, "validate", "flink-skill")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Must show OK marks for name and description.
	if !strings.Contains(combined, "✓") {
		t.Error("expected at least one ✓ check mark in output")
	}
	if !strings.Contains(combined, "name") {
		t.Error("expected 'name' field in output")
	}
	if !strings.Contains(combined, "description") {
		t.Error("expected 'description' field in output")
	}

	// Must not contain error marks.
	if strings.Contains(combined, "✗") {
		t.Errorf("expected no ✗ error marks, but output contained one:\n%s", combined)
	}
}

// TestValidateK8sDeployReview runs `skill-arena validate k8s-deploy-review`
// and asserts the example SKILL.md passes validation with no errors.
func TestValidateK8sDeployReview(t *testing.T) {
	binary := buildBinary(t)
	workDir := setupSkillDir(t, "k8s-deploy-review")

	stdout, stderr, exitCode := runCmd(t, binary, workDir, nil, "validate", "k8s-deploy-review")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if strings.Contains(combined, "✗") {
		t.Errorf("expected no ✗ error marks, but output contained one:\n%s", combined)
	}
}

// TestEvalRunNoAPIKey verifies that `skill-arena eval run` exits with code 1
// and reports a meaningful error when no API key is configured.
func TestEvalRunNoAPIKey(t *testing.T) {
	binary := buildBinary(t)
	workDir := setupSkillDir(t, "flink-skill")

	// Redirect HOME to a fresh temp dir so no real config is found.
	fakeHome, err := os.MkdirTemp("", "skill-arena-home-*")
	if err != nil {
		t.Fatalf("creating fake home: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(fakeHome) })

	env := []string{"HOME=" + fakeHome}
	stdout, stderr, exitCode := runCmd(t, binary, workDir, env, "eval", "run", "flink-skill")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode == 0 {
		t.Error("expected non-zero exit code when API key is not configured")
	}

	apiKeyMentioned := strings.Contains(strings.ToLower(combined), "api key") ||
		strings.Contains(strings.ToLower(combined), "api_key") ||
		strings.Contains(strings.ToLower(combined), "not configured")
	if !apiKeyMentioned {
		t.Errorf("expected output to mention 'API key' or 'not configured', got:\n%s", combined)
	}
}

// TestEvalRunGateEnforcement creates a minimal skill with only one eval case
// (missing required categories) and asserts that eval run is blocked.
func TestEvalRunGateEnforcement(t *testing.T) {
	binary := buildBinary(t)

	// Create a temporary working directory with a hand-crafted minimal skill.
	tmpDir, err := os.MkdirTemp("", "skill-arena-gate-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	skillDir := filepath.Join(tmpDir, ".claude", "skills", "minimal-skill")
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}

	skillMD := `---
name: minimal-skill
description: "A minimal test skill used only for gate enforcement testing. Trigger when user mentions: minimal, test."
---

## When to Activate

Use for testing only.

## Output Format

Plain text response.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}

	// evals.json with only a "core" case — missing "edge" and "error_diagnosis".
	type assertion struct {
		Type  string `json:"type"`
		Value string `json:"value,omitempty"`
	}
	type evalCase struct {
		ID             int         `json:"id"`
		Category       string      `json:"category"`
		Prompt         string      `json:"prompt"`
		ExpectedOutput string      `json:"expected_output"`
		Assertions     []assertion `json:"assertions"`
	}
	type evalFile struct {
		SkillName    string     `json:"skill_name"`
		SkillType    string     `json:"skill_type"`
		CodeLanguage string     `json:"code_language"`
		Evals        []evalCase `json:"evals"`
	}

	ef := evalFile{
		SkillName:    "minimal-skill",
		SkillType:    "coding",
		CodeLanguage: "go",
		Evals: []evalCase{
			{
				ID:             1,
				Category:       "core",
				Prompt:         "Write hello world in Go.",
				ExpectedOutput: "Simple main package with fmt.Println.",
				Assertions:     []assertion{{Type: "contains", Value: "Hello"}},
			},
		},
	}

	data, _ := json.MarshalIndent(ef, "", "  ")
	if err := os.WriteFile(filepath.Join(skillDir, "evals.json"), data, 0o644); err != nil {
		t.Fatalf("writing evals.json: %v", err)
	}

	// Redirect HOME so no real API key is found (we expect a gate error first).
	fakeHome, err := os.MkdirTemp("", "skill-arena-home-*")
	if err != nil {
		t.Fatalf("creating fake home: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(fakeHome) })

	// Write a config with a fake API key so the API key check passes and we
	// reach the category gate check.
	cfgDir := filepath.Join(fakeHome, ".skill-arena")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	cfgJSON := `{"api_key":"sk-fake-key-for-gate-test","api_base_url":"https://api.anthropic.com","default_model":"claude-haiku-4-5-20251001"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("writing config.json: %v", err)
	}

	env := []string{"HOME=" + fakeHome}
	stdout, stderr, exitCode := runCmd(t, binary, tmpDir, env, "eval", "run", "minimal-skill")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode == 0 {
		t.Error("expected non-zero exit code when required eval categories are missing")
	}

	mentionsCategories := strings.Contains(strings.ToLower(combined), "categor") ||
		strings.Contains(strings.ToLower(combined), "missing") ||
		strings.Contains(strings.ToLower(combined), "blocked")
	if !mentionsCategories {
		t.Errorf("expected output to mention missing categories or 'blocked', got:\n%s", combined)
	}
}

// ---- Integration tests (skipped unless SKILL_ARENA_API_KEY is set) ----

// integrationEnv returns the env slice and a fake HOME directory for
// integration tests. It writes a real config.json to the fake home so the
// subprocess reads it. Returns nil, "" if SKILL_ARENA_API_KEY is not set.
func integrationEnv(t *testing.T) (env []string, fakeHome string) {
	t.Helper()

	apiKey := os.Getenv("SKILL_ARENA_API_KEY")
	if apiKey == "" {
		return nil, ""
	}

	apiBase := os.Getenv("SKILL_ARENA_API_BASE_URL")
	if apiBase == "" {
		apiBase = "https://api.anthropic.com"
	}
	model := os.Getenv("SKILL_ARENA_MODEL")
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}

	fakeHome, err := os.MkdirTemp("", "skill-arena-int-home-*")
	if err != nil {
		t.Fatalf("creating fake home for integration test: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(fakeHome) })

	cfgDir := filepath.Join(fakeHome, ".skill-arena")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	cfg := map[string]string{
		"api_key":       apiKey,
		"api_base_url":  apiBase,
		"default_model": model,
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), data, 0o600); err != nil {
		t.Fatalf("writing integration config.json: %v", err)
	}

	return []string{"HOME=" + fakeHome}, fakeHome
}

// TestEvalRunFlink is an integration test that runs the full flink-skill eval
// against a real LLM API. It is skipped when SKILL_ARENA_API_KEY is not set.
func TestEvalRunFlink(t *testing.T) {
	env, _ := integrationEnv(t)
	if env == nil {
		t.Skip("SKILL_ARENA_API_KEY not set — skipping integration test")
	}

	binary := buildBinary(t)
	workDir := setupSkillDir(t, "flink-skill")

	stdout, stderr, exitCode := runCmd(t, binary, workDir, env, "eval", "run", "flink-skill")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d\noutput:\n%s", exitCode, combined)
	}

	// Output must include the inline summary section.
	if !strings.Contains(combined, "Summary") {
		t.Errorf("expected 'Summary' in output, got:\n%s", combined)
	}

	// History directory and report.md must be created inside the temp workdir.
	historyBase := filepath.Join(workDir, ".skill-arena", "history", "flink-skill")
	entries, err := os.ReadDir(historyBase)
	if err != nil {
		t.Fatalf("reading history dir %s: %v", historyBase, err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one run directory under %s", historyBase)
	}

	// Check that report.md exists and contains the expected header.
	runDir := filepath.Join(historyBase, entries[0].Name())
	reportPath := filepath.Join(runDir, "report.md")
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("reading report.md at %s: %v", reportPath, err)
	}
	if !strings.Contains(string(reportData), "# Eval Report") {
		t.Errorf("expected report.md to contain '# Eval Report', got:\n%s", string(reportData))
	}
}

// TestEvalRunK8sDeployReview is an integration test that runs the full
// k8s-deploy-review eval against a real LLM API.
func TestEvalRunK8sDeployReview(t *testing.T) {
	env, _ := integrationEnv(t)
	if env == nil {
		t.Skip("SKILL_ARENA_API_KEY not set — skipping integration test")
	}

	binary := buildBinary(t)
	workDir := setupSkillDir(t, "k8s-deploy-review")

	stdout, stderr, exitCode := runCmd(t, binary, workDir, env, "eval", "run", "k8s-deploy-review")
	combined := stdout + stderr

	t.Logf("stdout:\n%s", stdout)
	t.Logf("stderr:\n%s", stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d\noutput:\n%s", exitCode, combined)
	}

	if !strings.Contains(combined, "Summary") {
		t.Errorf("expected 'Summary' in output, got:\n%s", combined)
	}

	historyBase := filepath.Join(workDir, ".skill-arena", "history", "k8s-deploy-review")
	entries, err := os.ReadDir(historyBase)
	if err != nil {
		t.Fatalf("reading history dir %s: %v", historyBase, err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one run directory under %s", historyBase)
	}

	runDir := filepath.Join(historyBase, entries[0].Name())
	reportPath := filepath.Join(runDir, "report.md")
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("reading report.md at %s: %v", reportPath, err)
	}
	if !strings.Contains(string(reportData), "# Eval Report") {
		t.Errorf("expected report.md to contain '# Eval Report', got:\n%s", string(reportData))
	}
}
