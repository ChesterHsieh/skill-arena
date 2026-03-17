# Skill Arena

A local-first CLI tool for building and evaluating Claude Code skills.

The core question it answers: **"Does this skill actually improve LLM output quality vs. not having it?"**

---

## How it works

A Claude Code skill is a `SKILL.md` file that tells the AI how to behave in a specific domain. Skill Arena helps you:

1. **Scaffold** a new skill from a structured 4-question prompt
2. **Validate** it against the [Agent Skill SOP](./agent-skill-sop.md)
3. **Write eval cases** manually or have an LLM generate suggestions
4. **Run a with/without comparison** — two parallel API calls per eval, one with your skill injected and one without, then diff the outputs and score assertions

```
skill-arena init flink-skill       # scaffold
skill-arena validate flink-skill   # check SOP compliance
skill-arena eval generate flink-skill  # LLM suggests test cases
skill-arena eval add flink-skill   # or write them yourself
skill-arena eval run flink-skill   # measure the impact
skill-arena eval history flink-skill   # track improvement over time
```

---

## Install

### One-liner (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/install.sh | sh
```

Auto-detects your OS and architecture. If Go is installed, builds from source. Otherwise downloads a pre-built binary from GitHub Releases.

### If you have Go

```bash
go install github.com/ChesterHsieh/skill-arena@latest
```

Make sure `$(go env GOPATH)/bin` is in your `$PATH`.

### Build from source

```bash
git clone https://github.com/ChesterHsieh/skill-arena
cd skill-arena
go build -o skill-arena .
mv skill-arena /usr/local/bin/
```

### Uninstall

```bash
curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/uninstall.sh | sh
```

Removes the binary from `/usr/local/bin`, `~/.local/bin`, and `$(go env GOPATH)/bin`. Config (`~/.skill-arena/`) and eval history (`.skill-arena/`) are kept — remove manually if desired.

### Claude Code (no binary needed)

If you use Claude Code, clone this repo and open it as your project. The `/skill` slash command is included in `.claude/commands/skill.md` — Claude handles all commands natively by reading and writing files directly. No Go, no binary, no install.

```
/skill init my-skill
/skill validate flink-skill
/skill eval run flink-skill
/skill eval history flink-skill
```

Claude performs `validate` by checking SKILL.md rules itself, and `eval run` by running the with/without comparison using its own inference — no external API calls needed.

---

No runtime dependencies. Single static binary.

---

## Quick start

### 1. Configure your LLM

```bash
skill-arena config
```

Prompts for:
- **API base URL** — default `https://api.anthropic.com`. Use any OpenAI-compatible endpoint.
- **API key** — stored locally at `~/.skill-arena/config.json`, never sent anywhere except your configured API.
- **Default model** — default `claude-sonnet-4-6`.

### 2. Create a skill

```bash
cd your-project/    # must have a .claude/ directory or be a Claude Code project
skill-arena init my-skill
```

Answers 4 questions, then writes:

```
.claude/skills/my-skill/
├── SKILL.md          # the skill itself
├── evals.json        # test cases (3 template cases pre-filled)
└── references/       # drop supporting docs here
```

### 3. Edit and validate

```bash
skill-arena edit my-skill       # opens $EDITOR
skill-arena validate my-skill   # SOP compliance check
```

Validation output:

```
✓ name: present ("my-skill")
✓ description: 142 chars (min 50)
✓ description: contains trigger keywords
⚠ body: 312 lines (warning threshold: 400)
✗ missing: ## Output Format section

2 warnings, 1 error
```

### 4. Add eval cases

Either write them yourself:

```bash
skill-arena eval add my-skill
```

Or let the LLM suggest cases from your `SKILL.md`:

```bash
skill-arena eval generate my-skill
```

`eval run` is **blocked** until you have at least 3 cases covering all required categories for your skill type:

| Skill type | Required categories |
|------------|---------------------|
| `coding`   | `core`, `edge`, `error_diagnosis` |
| `workflow` | `core`, `partial`, `edge` |

### 5. Run the eval

```bash
skill-arena eval run my-skill
```

Output:

```
Running 3 eval cases against claude-sonnet-4-6 (api.anthropic.com)
Each case: 2 parallel requests (with skill / without skill)

Case 1/3 [core]: "Write a Flink job that reads from Kafka..."
  WITH    → 847 tokens  [3/4 assertions passed]
  WITHOUT → 612 tokens  [1/4 assertions passed]

Case 2/3 [error_diagnosis]: "My checkpoint keeps timing out..."
  WITH    → 523 tokens  [2/2 assertions passed]
  WITHOUT → 398 tokens  [0/2 assertions passed]

Case 3/3 [edge]: "Implement late event handling with side output..."
  WITH    → 634 tokens  [2/3 assertions passed]
  WITHOUT → 501 tokens  [1/3 assertions passed]

────────────────────────────────────────
Summary  WITH:    7/9 assertions passed (78%)
         WITHOUT: 2/9 assertions passed (22%)
         Impact:  +56pp assertion improvement
         Avg token delta: +198 tokens with skill
────────────────────────────────────────
Full report: .skill-arena/history/flink-skill/2026-03-16T10-00-00/report.md
```

The full report is saved as a Markdown file with complete WITH/WITHOUT responses, diffs, and per-assertion results.

### 6. Track improvement

```bash
skill-arena eval history my-skill

Run history for: my-skill
  2026-03-16 10:00  7/9 passed (78% WITH, 22% WITHOUT)  +56pp  [current]
  2026-03-15 14:32  5/9 passed (56% WITH, 22% WITHOUT)  +34pp
  2026-03-14 09:11  3/9 passed (33% WITH, 22% WITHOUT)  +11pp

Trend: improving ↑
```

---

## Skill types

### Coding (`"skill_type": "coding"`)

For skills that produce executable code. Adds code-specific assertions:

| Assertion | Behavior |
|-----------|----------|
| `syntax_valid` | Parses code blocks from the response. Go: uses `go/parser`. Others: balanced-brace heuristic. |
| `code_style` | Runs your configured linter binary against extracted code blocks. |
| `compiles` | Writes code to a temp dir and compiles it. Supports Go and Python. |
| `contains_pattern` | Regex match on code block content. |

### Workflow (`"skill_type": "workflow"`)

For skills that guide multi-step processes (reviews, debugging, migrations, etc.). Adds step-tracking assertions:

| Assertion | Behavior |
|-----------|----------|
| `step_present` | Response mentions a specific step or concept (case-insensitive). |
| `step_order` | Step A appears before step B in the response. |
| `all_steps_covered` | All listed steps appear somewhere in the response. |
| `no_skipped_gate` | A required checkpoint phrase is present before the response ends. |

Both types support `contains`, `not_contains`, and `quality` (LLM judge with a rubric).

---

## Isolation model

When you run `eval run my-skill`, only `my-skill` is injected into context — no other skills from your `.claude/skills/` directory are included. This isolates the measurement so the result reflects the skill under test, not your full skill stack.

```
WITH skill:    system = base prompt + ONLY my-skill/SKILL.md
WITHOUT skill: system = base prompt only
```

---

## BYOM (Bring Your Own Model)

`skill-arena config` accepts any API base URL:

```
# Anthropic (default)
API base URL: https://api.anthropic.com
Model: claude-sonnet-4-6

# OpenAI
API base URL: https://api.openai.com/v1
Model: gpt-4o

# Local (Ollama, LM Studio, etc.)
API base URL: http://localhost:11434/v1
Model: llama3.2
```

Auto-detects Anthropic vs OpenAI-compatible format from the URL. Config stored at `~/.skill-arena/config.json`.

---

## Examples

Two complete examples are included in [`examples/`](./examples/):

### `examples/flink-skill/` — coding skill

Apache Flink DataStream/Table API assistant. Demonstrates:
- Trigger keywords for domain-specific activation
- Three workflow paths (code gen, architecture, error diagnosis)
- `syntax_valid`, `contains_pattern`, `all_steps_covered`, and `quality` assertions
- `references/datastream-api.md` for window types, watermark strategies, state backends

```bash
cp -r examples/flink-skill .claude/skills/
skill-arena validate flink-skill
skill-arena eval run flink-skill
```

### `examples/k8s-deploy-review/` — workflow skill

Kubernetes deployment manifest review across 4 gates: Security → Reliability → Observability → Summary. Demonstrates:
- Multi-gate ordered workflow
- `step_order`, `all_steps_covered`, and `quality` assertions
- Handling partial reviews (the `partial` eval category)
- `references/checklist.md` with field-level K8s checks

```bash
cp -r examples/k8s-deploy-review .claude/skills/
skill-arena validate k8s-deploy-review
skill-arena eval run k8s-deploy-review
```

---

## E2E tests

```bash
# Unit-level (no API key needed)
go test -tags e2e ./e2e/

# Integration (requires API key)
SKILL_ARENA_API_KEY=sk-ant-... go test -tags e2e -v ./e2e/

# Custom endpoint
SKILL_ARENA_API_KEY=sk-... \
SKILL_ARENA_API_BASE_URL=https://api.openai.com/v1 \
SKILL_ARENA_MODEL=gpt-4o \
go test -tags e2e -v ./e2e/
```

See [`e2e/README.md`](./e2e/README.md) for the full test coverage table.

---

## File layout

```
.claude/skills/<name>/
├── SKILL.md          # required — name + description frontmatter + body
├── evals.json        # required to run evals — test cases + assertions
└── references/       # optional — supporting docs loaded on demand

~/.skill-arena/
└── config.json       # API key, base URL, default model

.skill-arena/history/<name>/<timestamp>/
├── report.md         # full eval report with diffs and assertion results
└── results.json      # structured results for tooling
```

---

## Commands reference

```
skill-arena init <name>               Scaffold a new skill (4 SOP questions + LLM structuring)
skill-arena edit <name>               Open SKILL.md in $EDITOR
skill-arena validate <name>           Check SOP compliance (6 rules)
skill-arena config                    Set API endpoint, key, and model

skill-arena eval add <name>           Add an eval case interactively
skill-arena eval generate <name>      LLM suggests eval cases from SKILL.md
skill-arena eval run <name>           Run with/without comparison
skill-arena eval history <name>       Show past eval runs and trend
```

Full reference: [docs/COMMANDS.md](./docs/COMMANDS.md) · [docs/CONFIG.md](./docs/CONFIG.md) · [docs/CONTRIBUTING.md](./docs/CONTRIBUTING.md)
