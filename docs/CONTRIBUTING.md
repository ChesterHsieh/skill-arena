# Contributing

## Prerequisites

- Go 1.22+
- An Anthropic or OpenAI-compatible API key (for integration tests and `skill-arena init` LLM generation)

## Setup

```bash
git clone https://github.com/ChesterHsieh/skill-arena
cd skill-arena
go mod tidy
go build ./...
```

## Commands

<!-- AUTO-GENERATED from cobra command definitions -->

| Command | Description |
|---------|-------------|
| `go build ./...` | Build all packages |
| `go test ./...` | Run unit tests |
| `go test -tags e2e ./e2e/` | Run e2e tests (no API key needed) |
| `go vet ./...` | Run static analysis |
| `go run main.go --help` | Run CLI without installing |

<!-- END AUTO-GENERATED -->

## Project layout

```
cmd/              CLI layer (cobra commands)
  eval/           eval subcommands (add, generate, run, history)
internal/
  config/         config load/save (~/.skill-arena/config.json)
  skill/          SKILL.md + evals.json IO, scaffold, validate
  eval/           eval runner, assertions, report, history
  llm/            HTTP client (Anthropic + OpenAI-compatible)
e2e/              end-to-end tests (build tag: e2e)
examples/         example skills (flink-skill, k8s-deploy-review)
```

## Testing

**Unit tests** — no API key required:
```bash
go test ./...
```

**E2E tests (unit level)** — exercises the built binary, no API key:
```bash
go test -tags e2e ./e2e/
```

**E2E tests (integration)** — requires a real API key:
```bash
SKILL_ARENA_API_KEY=sk-ant-... go test -tags e2e -v ./e2e/

# Custom endpoint (OpenAI, local, etc.)
SKILL_ARENA_API_KEY=sk-... \
SKILL_ARENA_API_BASE_URL=https://api.openai.com/v1 \
SKILL_ARENA_MODEL=gpt-4o \
go test -tags e2e -v ./e2e/
```

## Adding a new CLI command

1. Create `cmd/<name>.go` with a `cobra.Command`
2. Register it in `cmd/root.go` `init()`
3. Add a unit test or e2e test in `e2e/e2e_test.go`

## Adding a new assertion type

1. Add the type constant and handling to `internal/eval/assert.go`
2. Document it in `RFP.md` §10
3. Add an example case to `examples/flink-skill/evals.json` or `examples/k8s-deploy-review/evals.json`

## Releasing

Releases are automated via GoReleaser. To cut a new version:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions picks up the tag and publishes binaries for all 5 targets (Linux amd64/arm64, macOS amd64/arm64, Windows amd64).

## PR checklist

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go test -tags e2e ./e2e/` passes
- [ ] `go vet ./...` passes
- [ ] New commands/assertions documented in `README.md` and `RFP.md`
