# E2E Tests

## Overview

The `e2e/` package contains end-to-end tests for `skill-arena`. Tests use the `e2e` build tag and compile + run the real binary against example skills in `examples/`.

## Running Unit-Level E2E Tests

These tests do not require an API key. They validate CLI behaviour for commands that either do not call an LLM (e.g., `validate`) or fail fast before making a network request (e.g., `eval run` with no key configured).

```bash
go test -tags e2e -v ./e2e/
```

To run a specific test:

```bash
go test -tags e2e -run TestValidateFlink -v ./e2e/
go test -tags e2e -run TestValidateK8sDeployReview -v ./e2e/
go test -tags e2e -run TestEvalRunNoAPIKey -v ./e2e/
go test -tags e2e -run TestEvalRunGateEnforcement -v ./e2e/
```

## Running Integration Tests

Integration tests call a real LLM API and are skipped unless `SKILL_ARENA_API_KEY` is set.

```bash
export SKILL_ARENA_API_KEY=sk-ant-...
export SKILL_ARENA_API_BASE_URL=https://api.anthropic.com   # optional, this is the default
export SKILL_ARENA_MODEL=claude-haiku-4-5-20251001          # optional, this is the default

go test -tags e2e -run TestEvalRunFlink -v -timeout 5m ./e2e/
go test -tags e2e -run TestEvalRunK8sDeployReview -v -timeout 5m ./e2e/
```

The integration tests write a temporary `~/.skill-arena/config.json` to a freshly-created temp `HOME` directory and set `HOME` for the subprocess. Your real config is never read or modified.

## How It Works

1. `buildBinary` compiles the project root into a temp directory. The result is cached for the duration of the test run (built once, shared across all tests).
2. `setupSkillDir` creates a fresh temp working directory and copies the requested example from `examples/<name>/` into `.claude/skills/<name>/` — exactly where `skill-arena` expects to find a skill.
3. `runCmd` executes the binary with the given args and captures stdout, stderr, and the exit code for assertions.

## Test Coverage

| Test | Requires API key | What it checks |
|------|-----------------|----------------|
| `TestValidateFlink` | No | `validate flink-skill` exits 0, shows ✓ marks, no errors |
| `TestValidateK8sDeployReview` | No | `validate k8s-deploy-review` exits 0, no errors |
| `TestEvalRunNoAPIKey` | No | `eval run` exits 1, output mentions API key |
| `TestEvalRunGateEnforcement` | No | `eval run` blocked when required categories are missing |
| `TestEvalRunFlink` | Yes | Full eval run completes, report.md written |
| `TestEvalRunK8sDeployReview` | Yes | Full eval run completes, report.md written |
