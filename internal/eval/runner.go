package eval

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/ChesterHsieh/skill-arena/internal/config"
	"github.com/ChesterHsieh/skill-arena/internal/llm"
	"github.com/ChesterHsieh/skill-arena/internal/skill"
)

// RunResult holds the comparison result for one eval case.
type RunResult struct {
	EvalID       int
	Category     string
	Prompt       string
	WithSkill    SideResult
	WithoutSkill SideResult
	Diff         string
}

// SideResult holds the outcome of one side (with or without skill) of an eval.
type SideResult struct {
	Response     string
	InputTokens  int
	OutputTokens int
	Assertions   []AssertionResult
	PassCount    int
	TotalCount   int
}

const baseSystemPrompt = "You are a helpful AI assistant."

// Run executes the full with/without eval for a skill and returns all results.
func Run(ctx context.Context, skillDir string, cfg *config.Config) ([]RunResult, error) {
	skillContent, err := skill.ReadSkillContent(skillDir)
	if err != nil {
		return nil, fmt.Errorf("reading skill content: %w", err)
	}

	ef, err := skill.ReadEvalFile(skillDir)
	if err != nil {
		return nil, fmt.Errorf("reading eval file: %w", err)
	}

	// Gate: check required categories
	if err := checkRequiredCategories(ef); err != nil {
		return nil, err
	}

	client := llm.NewClient(cfg)
	dmp := diffmatchpatch.New()

	results := make([]RunResult, len(ef.Evals))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var runErr error

	for i, evalCase := range ef.Evals {
		wg.Add(1)
		go func(idx int, ec skill.EvalCase) {
			defer wg.Done()

			result, err := runEvalCase(ctx, ec, skillContent, ef, client, dmp)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				runErr = fmt.Errorf("eval case %d: %w", ec.ID, err)
				return
			}
			results[idx] = result
		}(i, evalCase)
	}

	wg.Wait()
	if runErr != nil {
		return nil, runErr
	}

	return results, nil
}

// runEvalCase runs both the with-skill and without-skill requests for one eval case.
func runEvalCase(
	ctx context.Context,
	ec skill.EvalCase,
	skillContent string,
	ef *skill.EvalFile,
	client *llm.Client,
	dmp *diffmatchpatch.DiffMatchPatch,
) (RunResult, error) {
	withSystemPrompt := baseSystemPrompt + "\n\n" + skillContent
	withoutSystemPrompt := baseSystemPrompt

	type sideResponse struct {
		resp *llm.CompletionResponse
		err  error
	}

	withCh := make(chan sideResponse, 1)
	withoutCh := make(chan sideResponse, 1)

	go func() {
		resp, err := client.Complete(ctx, withSystemPrompt, ec.Prompt)
		withCh <- sideResponse{resp, err}
	}()
	go func() {
		resp, err := client.Complete(ctx, withoutSystemPrompt, ec.Prompt)
		withoutCh <- sideResponse{resp, err}
	}()

	withResp := <-withCh
	withoutResp := <-withoutCh

	if withResp.err != nil {
		return RunResult{}, fmt.Errorf("with-skill request failed: %w", withResp.err)
	}
	if withoutResp.err != nil {
		return RunResult{}, fmt.Errorf("without-skill request failed: %w", withoutResp.err)
	}

	withSide := evaluateSide(ctx, withResp.resp, ec.Assertions, ef.CodeLanguage, ef.LinterCmd, client)
	withoutSide := evaluateSide(ctx, withoutResp.resp, ec.Assertions, ef.CodeLanguage, ef.LinterCmd, client)

	// Generate unified diff
	diffs := dmp.DiffMain(withoutResp.resp.Content, withResp.resp.Content, false)
	dmp.DiffCleanupSemantic(diffs)
	patches := dmp.PatchMake(withoutResp.resp.Content, diffs)
	diffText := dmp.PatchToText(patches)

	return RunResult{
		EvalID:       ec.ID,
		Category:     ec.Category,
		Prompt:       ec.Prompt,
		WithSkill:    withSide,
		WithoutSkill: withoutSide,
		Diff:         diffText,
	}, nil
}

// evaluateSide scores assertions for one side of an eval case.
func evaluateSide(
	ctx context.Context,
	resp *llm.CompletionResponse,
	assertions []skill.Assertion,
	codeLanguage string,
	linterCmd string,
	client *llm.Client,
) SideResult {
	side := SideResult{
		Response:     resp.Content,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	}

	for _, a := range assertions {
		result := EvaluateAssertion(ctx, a, resp.Content, codeLanguage, linterCmd, client)
		side.Assertions = append(side.Assertions, result)
		side.TotalCount++
		if result.Passed {
			side.PassCount++
		}
	}

	return side
}

// checkRequiredCategories verifies that all required categories are present in the eval file.
func checkRequiredCategories(ef *skill.EvalFile) error {
	categories := map[string]bool{}
	for _, ec := range ef.Evals {
		categories[ec.Category] = true
	}

	var required []string
	switch ef.SkillType {
	case skill.SkillTypeCoding:
		required = []string{"core", "edge", "error_diagnosis"}
	default: // workflow
		required = []string{"core", "partial", "edge"}
	}

	var missing []string
	for _, cat := range required {
		if !categories[cat] {
			missing = append(missing, cat)
		}
	}

	if len(missing) > 0 {
		hint := fmt.Sprintf("Run 'skill-arena eval add %s' or 'skill-arena eval generate %s' to add missing cases.", ef.SkillName, ef.SkillName)
		return fmt.Errorf(
			"eval run blocked: missing required categories for %s skill: %s\n\nRequired: %s\n%s",
			ef.SkillType,
			strings.Join(missing, ", "),
			strings.Join(required, ", "),
			hint,
		)
	}

	if len(ef.Evals) < 3 {
		return fmt.Errorf(
			"eval run blocked: need at least 3 eval cases, found %d\n\nRun 'skill-arena eval add %s' to add more cases.",
			len(ef.Evals), ef.SkillName,
		)
	}

	return nil
}
