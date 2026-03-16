package eval

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ChesterHsieh/skill-arena/internal/llm"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

// AssertionResult holds the outcome of evaluating a single assertion.
type AssertionResult struct {
	Type    string
	Passed  bool
	Message string
}

// EvaluateAssertion checks a single assertion against the given response.
func EvaluateAssertion(
	ctx context.Context,
	a skill.Assertion,
	response string,
	codeLanguage string,
	linterCmd string,
	llmClient *llm.Client,
) AssertionResult {
	switch a.Type {
	case "contains":
		passed := strings.Contains(response, a.Value)
		msg := fmt.Sprintf("expected %q in response", a.Value)
		if passed {
			msg = fmt.Sprintf("found %q", a.Value)
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "not_contains":
		passed := !strings.Contains(response, a.Value)
		msg := fmt.Sprintf("%q must not appear in response", a.Value)
		if passed {
			msg = fmt.Sprintf("%q not found (correct)", a.Value)
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "contains_pattern":
		matched, err := regexp.MatchString(a.Pattern, response)
		if err != nil {
			return AssertionResult{Type: a.Type, Passed: false, Message: fmt.Sprintf("invalid regex %q: %v", a.Pattern, err)}
		}
		msg := fmt.Sprintf("pattern %q not matched", a.Pattern)
		if matched {
			msg = fmt.Sprintf("pattern %q matched", a.Pattern)
		}
		return AssertionResult{Type: a.Type, Passed: matched, Message: msg}

	case "step_present":
		lower := strings.ToLower(response)
		passed := strings.Contains(lower, strings.ToLower(a.Value))
		msg := fmt.Sprintf("step %q not found in response", a.Value)
		if passed {
			msg = fmt.Sprintf("step %q found", a.Value)
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "step_order":
		lower := strings.ToLower(response)
		beforeIdx := strings.Index(lower, strings.ToLower(a.Before))
		afterIdx := strings.Index(lower, strings.ToLower(a.After))
		if beforeIdx == -1 {
			return AssertionResult{Type: a.Type, Passed: false, Message: fmt.Sprintf("%q not found in response", a.Before)}
		}
		if afterIdx == -1 {
			return AssertionResult{Type: a.Type, Passed: false, Message: fmt.Sprintf("%q not found in response", a.After)}
		}
		passed := beforeIdx < afterIdx
		msg := fmt.Sprintf("%q appears before %q (correct order)", a.Before, a.After)
		if !passed {
			msg = fmt.Sprintf("%q should appear before %q but does not", a.Before, a.After)
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "all_steps_covered":
		lower := strings.ToLower(response)
		var missing []string
		for _, step := range a.Steps {
			if !strings.Contains(lower, strings.ToLower(step)) {
				missing = append(missing, step)
			}
		}
		passed := len(missing) == 0
		msg := fmt.Sprintf("all %d steps found", len(a.Steps))
		if !passed {
			msg = fmt.Sprintf("missing steps: %s", strings.Join(missing, ", "))
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "no_skipped_gate":
		passed := strings.Contains(response, a.Value)
		msg := fmt.Sprintf("gate phrase %q present (not skipped)", a.Value)
		if !passed {
			msg = fmt.Sprintf("gate phrase %q missing — checkpoint was skipped", a.Value)
		}
		return AssertionResult{Type: a.Type, Passed: passed, Message: msg}

	case "syntax_valid":
		return assertSyntaxValid(response, codeLanguage)

	case "code_style":
		return assertCodeStyle(response, codeLanguage, linterCmd)

	case "compiles":
		return assertCompiles(response, codeLanguage)

	case "quality":
		return assertQuality(ctx, a, response, llmClient)

	default:
		return AssertionResult{Type: a.Type, Passed: false, Message: fmt.Sprintf("unknown assertion type %q", a.Type)}
	}
}

// assertSyntaxValid extracts code blocks and checks syntax.
func assertSyntaxValid(response, codeLanguage string) AssertionResult {
	blocks := extractCodeBlocks(response)
	if len(blocks) == 0 {
		return AssertionResult{Type: "syntax_valid", Passed: false, Message: "no code blocks found in response"}
	}

	for i, block := range blocks {
		switch strings.ToLower(codeLanguage) {
		case "go":
			fset := token.NewFileSet()
			_, err := parser.ParseFile(fset, "", block, parser.AllErrors)
			if err != nil {
				return AssertionResult{Type: "syntax_valid", Passed: false, Message: fmt.Sprintf("syntax error in block %d: %v", i+1, err)}
			}
		default:
			// Heuristic: check balanced braces/brackets
			if err := checkBalanced(block); err != nil {
				return AssertionResult{Type: "syntax_valid", Passed: false, Message: fmt.Sprintf("unbalanced brackets in block %d: %v", i+1, err)}
			}
		}
	}
	return AssertionResult{Type: "syntax_valid", Passed: true, Message: fmt.Sprintf("%d code block(s) parsed without errors", len(blocks))}
}

// assertCodeStyle runs a linter on extracted code blocks.
func assertCodeStyle(response, codeLanguage, linterCmd string) AssertionResult {
	if linterCmd == "" {
		return AssertionResult{Type: "code_style", Passed: true, Message: "skipped (no linter_cmd configured)"}
	}

	blocks := extractCodeBlocks(response)
	if len(blocks) == 0 {
		return AssertionResult{Type: "code_style", Passed: false, Message: "no code blocks found to lint"}
	}

	ext := extensionFor(codeLanguage)
	for i, block := range blocks {
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("skill-arena-lint-*%s", ext))
		if err != nil {
			return AssertionResult{Type: "code_style", Passed: false, Message: fmt.Sprintf("cannot create temp file: %v", err)}
		}
		name := tmpFile.Name()
		defer os.Remove(name)

		if _, err := tmpFile.WriteString(block); err != nil {
			tmpFile.Close()
			return AssertionResult{Type: "code_style", Passed: false, Message: fmt.Sprintf("cannot write temp file: %v", err)}
		}
		tmpFile.Close()

		cmd := exec.Command("sh", "-c", linterCmd+" "+name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return AssertionResult{Type: "code_style", Passed: false, Message: fmt.Sprintf("linter failed on block %d: %s", i+1, out.String())}
		}
	}
	return AssertionResult{Type: "code_style", Passed: true, Message: "all code blocks passed linter"}
}

// assertCompiles extracts code and attempts to compile it.
func assertCompiles(response, codeLanguage string) AssertionResult {
	blocks := extractCodeBlocks(response)
	if len(blocks) == 0 {
		return AssertionResult{Type: "compiles", Passed: false, Message: "no code blocks found in response"}
	}

	switch strings.ToLower(codeLanguage) {
	case "go":
		return compileGo(blocks)
	case "python", "python3":
		return compilePython(blocks)
	default:
		return AssertionResult{Type: "compiles", Passed: true, Message: fmt.Sprintf("skipped (compile check not supported for %q)", codeLanguage)}
	}
}

func compileGo(blocks []string) AssertionResult {
	tmpDir, err := os.MkdirTemp("", "skill-arena-compile-*")
	if err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("cannot create temp dir: %v", err)}
	}
	defer os.RemoveAll(tmpDir)

	// Write a minimal go.mod so we can run go build
	goMod := "module skillcheck\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("cannot write go.mod: %v", err)}
	}

	combined := strings.Join(blocks, "\n\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(combined), 0o644); err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("cannot write main.go: %v", err)}
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = tmpDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("go build failed: %s", out.String())}
	}
	return AssertionResult{Type: "compiles", Passed: true, Message: "go build succeeded"}
}

func compilePython(blocks []string) AssertionResult {
	tmpFile, err := os.CreateTemp("", "skill-arena-compile-*.py")
	if err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("cannot create temp file: %v", err)}
	}
	name := tmpFile.Name()
	defer os.Remove(name)

	combined := strings.Join(blocks, "\n\n")
	if _, err := tmpFile.WriteString(combined); err != nil {
		tmpFile.Close()
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("cannot write temp file: %v", err)}
	}
	tmpFile.Close()

	cmd := exec.Command("python3", "-m", "py_compile", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return AssertionResult{Type: "compiles", Passed: false, Message: fmt.Sprintf("python3 compile failed: %s", out.String())}
	}
	return AssertionResult{Type: "compiles", Passed: true, Message: "python3 -m py_compile succeeded"}
}

// assertQuality uses an LLM judge to evaluate response quality.
func assertQuality(ctx context.Context, a skill.Assertion, response string, llmClient *llm.Client) AssertionResult {
	if llmClient == nil {
		return AssertionResult{Type: "quality", Passed: false, Message: "no LLM client available for quality assertion"}
	}

	userPrompt := fmt.Sprintf(
		"Evaluate this response. Reply with exactly PASS or FAIL on the first line, then one sentence reason.\n\nRubric: %s\n\nResponse to evaluate:\n%s",
		a.Rubric, response,
	)

	resp, err := llmClient.Complete(ctx, "You are an evaluator.", userPrompt)
	if err != nil {
		return AssertionResult{Type: "quality", Passed: false, Message: fmt.Sprintf("LLM judge call failed: %v", err)}
	}

	lines := strings.SplitN(strings.TrimSpace(resp.Content), "\n", 2)
	verdict := strings.TrimSpace(lines[0])
	reason := ""
	if len(lines) > 1 {
		reason = strings.TrimSpace(lines[1])
	}

	passed := strings.HasPrefix(strings.ToUpper(verdict), "PASS")
	msg := verdict
	if reason != "" {
		msg = fmt.Sprintf("%s — %s", verdict, reason)
	}
	return AssertionResult{Type: "quality", Passed: passed, Message: msg}
}

// extractCodeBlocks extracts the content of fenced code blocks (```...```).
func extractCodeBlocks(response string) []string {
	re := regexp.MustCompile("(?s)```[a-z]*\n(.*?)```")
	matches := re.FindAllStringSubmatch(response, -1)
	var blocks []string
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, m[1])
		}
	}
	return blocks
}

// checkBalanced checks that braces, brackets, and parens are balanced.
func checkBalanced(code string) error {
	stack := []rune{}
	pairs := map[rune]rune{'}': '{', ']': '[', ')': '('}
	opens := map[rune]bool{'{': true, '[': true, '(': true}

	inString := false
	stringChar := rune(0)
	for _, ch := range code {
		if inString {
			if ch == stringChar {
				inString = false
			}
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			inString = true
			stringChar = ch
			continue
		}
		if opens[ch] {
			stack = append(stack, ch)
		} else if open, ok := pairs[ch]; ok {
			if len(stack) == 0 || stack[len(stack)-1] != open {
				return fmt.Errorf("unexpected %c", ch)
			}
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) > 0 {
		return fmt.Errorf("unclosed %c", stack[len(stack)-1])
	}
	return nil
}

// extensionFor returns a file extension for a given language.
func extensionFor(lang string) string {
	switch strings.ToLower(lang) {
	case "go":
		return ".go"
	case "python", "python3":
		return ".py"
	case "javascript", "js":
		return ".js"
	case "typescript", "ts":
		return ".ts"
	case "java":
		return ".java"
	case "ruby":
		return ".rb"
	default:
		return ".txt"
	}
}
