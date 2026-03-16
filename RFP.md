# Skill Arena — RFP v0.3

> **KISS principle applies throughout.** Target audience: AI engineers who already understand Claude Code skills.

---

## 1. Problem

Building a Claude Code skill today requires:
- Manually writing `SKILL.md` with no structural guidance
- Running two separate Claude sessions to compare with/without skill output
- No standardized way to write `evals.json` or measure if the skill actually works
- No feedback loop — you ship a skill without knowing if it helps

**Root cause:** The [Agent Skill SOP](./agent-skill-sop.md) is solid, but the workflow is entirely manual. There is no tooling.

---

## 2. Core Question This Product Must Answer

> **"Does this skill actually improve output quality vs. not having it?"**

Every design decision should serve this question. If a feature doesn't help answer it, cut it.

---

## 3. What We're Building

**A local-first CLI tool** — single binary, zero runtime dependencies, no npm. You run `skill-arena` from your project directory and it scaffolds, validates, and evals your skills from the terminal.

Think: `git` for Claude Code skills. Not a web app, not a SaaS.

---

## 4. Target Users

| User | Context |
|------|---------|
| AI engineer | Building domain-specific skills for their team's Claude setup |
| Platform developer | Maintaining a skill library across multiple projects |
| Power Claude Code user | Wants to verify their skills actually do something |

Assumed knowledge: knows what `SKILL.md` is, uses Claude Code daily.

---

## 5. CLI Commands (MVP)

```
skill-arena init <name>           # Scaffold a new skill (runs 4 SOP questions interactively)
skill-arena edit <name>           # Open SKILL.md in $EDITOR
skill-arena validate <name>       # Check SKILL.md against SOP rules
skill-arena eval add <name>       # Add an eval case interactively
skill-arena eval generate <name>  # LLM generates suggested eval cases from SKILL.md
skill-arena eval run <name>       # Run with/without comparison (blocked if < 3 eval cases)
skill-arena eval history <name>   # Show past eval run results
skill-arena config                # Set API endpoint, API key, default model
```

All commands operate on `.claude/skills/` in the current working directory (project-level).

**Eval gate:** `eval run` is blocked until the skill has at least 3 eval cases covering all required categories for its skill type (see §8). This is not optional.

---

## 6. Core Flows

### Flow A — Scaffold a Skill (`init`)

Interactive prompts based on the 4 SOP questions:

```
$ skill-arena init flink-skill

? What should the AI do with this skill?
> Help write Apache Flink DataStream / Table API code and diagnose checkpoint issues

? When should it trigger? (list keywords / user intent)
> Flink, stream processing, Kafka source, watermark, backpressure, checkpoint, state backend

? What is the expected output format?
> Executable Java/Python code + architecture explanation + caveats

? Is this a code-generation skill? (yes/no)
> yes

✓ Created .claude/skills/flink-skill/SKILL.md
✓ Created .claude/skills/flink-skill/evals.json  (template with 3 empty cases)
✓ Created .claude/skills/flink-skill/references/  (empty)
```

### Flow B — Validate (`validate`)

```
$ skill-arena validate flink-skill

✓ name: present
✓ description: 142 chars (min 50)
✓ description: contains trigger keywords (Flink, watermark, checkpoint)
⚠ body: 312 lines (warning threshold: 400)
✗ missing: ## Output Format section
  → add a section describing expected response structure

2 warnings, 1 error
```

### Flow C — Run Eval (`eval run`)

```
$ skill-arena eval run flink-skill

Running 3 eval cases against claude-sonnet-4-6 (api.anthropic.com)
Each case: 2 parallel requests (with skill / without skill)

Case 1/3: "Write a Flink job that reads from Kafka..."
  WITH    → 847 tokens  [assertions: 3/4 passed]
  WITHOUT → 612 tokens  [assertions: 1/4 passed]
  Delta diff saved → .skill-arena/history/flink-skill/2026-03-16T10:00:00/case-1.diff

Case 2/3: "My Flink job keeps hitting checkpoint timeout..."
  WITH    → 523 tokens  [assertions: 2/2 passed]
  WITHOUT → 398 tokens  [assertions: 0/2 passed]
  Delta diff saved → ...

Case 3/3: "How do I implement watermarks for late events..."
  WITH    → 634 tokens  [assertions: 2/3 passed]
  WITHOUT → 501 tokens  [assertions: 1/3 passed]

────────────────────────────────────────
Summary  WITH: 7/9 assertions passed (78%)
         WITHOUT: 2/9 assertions passed (22%)
         Skill impact: +56pp assertion improvement
         Avg token delta: +198 tokens with skill (cost tradeoff)
────────────────────────────────────────

Diffs written to .skill-arena/history/flink-skill/2026-03-16T10:00:00/
Run 'skill-arena eval history flink-skill' to browse
```

Diff output uses standard unified diff format — viewable with any `$DIFFTOOL` or `delta`.

### Flow D — History (`eval history`)

```
$ skill-arena eval history flink-skill

Run history for: flink-skill
  2026-03-16 10:00  7/9 passed (78% WITH, 22% WITHOUT)  +56pp  [current]
  2026-03-15 14:32  5/9 passed (56% WITH, 22% WITHOUT)  +34pp
  2026-03-14 09:11  3/9 passed (33% WITH, 22% WITHOUT)  +11pp

Trend: improving ↑
```

---

## 7. Skill Types & Eval Requirements

Declared in `evals.json` under `"skill_type"`. The type gates which assertion categories are required and unlocks type-specific verifiers.

### Type A — Coding Skill

> Skill that produces executable code as output (functions, classes, configs, scripts).

**Required eval categories (at least one case each):**

| Category | Purpose |
|----------|---------|
| `core` | Generates correct code for the most common use case |
| `edge` | Handles an unusual or complex variant |
| `error_diagnosis` | Reads broken code / error logs and explains the fix |

**Type-specific assertions available:**

| Assertion | Behavior |
|-----------|---------|
| `syntax_valid` | Extracts code blocks from response, runs language-appropriate parser (Go: `go/parser`, Python: `ast.parse`, JS: `acorn`) |
| `code_style` | Runs a linter on extracted code blocks against a configurable ruleset — user provides linter binary path in config (e.g., `eslint`, `ruff`, `golangci-lint`) |
| `compiles` | Actually compiles the extracted code in a temp dir (language must be configured; Go and Python supported in v1) |
| `contains_pattern` | Regex match on the code block content (e.g., must contain `try/catch`, must use `async/await`) |

**Example `evals.json` for a coding skill:**
```json
{
  "skill_name": "flink-skill",
  "skill_type": "coding",
  "code_language": "java",
  "linter": "checkstyle",
  "evals": [
    {
      "id": 1,
      "category": "core",
      "prompt": "Write a Flink job that reads from Kafka and computes a 5-min tumbling window sum",
      "expected_output": "Complete DataStream API code with KafkaSource, keyBy, window, aggregate, KafkaSink",
      "assertions": [
        { "type": "syntax_valid" },
        { "type": "code_style" },
        { "type": "contains_pattern", "pattern": "TumblingEventTimeWindows|TumblingProcessingTimeWindows" }
      ]
    }
  ]
}
```

---

### Type B — Workflow / Process Skill

> Skill that guides the AI through a multi-step job: debugging, review, planning, migration, etc.

**Required eval categories (at least one case each):**

| Category | Purpose |
|----------|---------|
| `core` | Full happy-path execution of the workflow |
| `partial` | Skill is invoked mid-way; must pick up correctly |
| `edge` | Ambiguous input or missing info; must handle gracefully |

**Type-specific assertions available:**

| Assertion | Behavior |
|-----------|---------|
| `step_present` | Checks that a specific step/phase appears in the response (string or regex) |
| `step_order` | Verifies step A appears before step B in the response |
| `all_steps_covered` | Given a list of required steps, all must appear somewhere in the response |
| `no_skipped_gate` | Response must not proceed past a defined checkpoint without acknowledging it |
| `quality` | LLM judge: send response + rubric to model, get pass/fail with reason |

**Example `evals.json` for a workflow skill:**
```json
{
  "skill_name": "pr-review-skill",
  "skill_type": "workflow",
  "evals": [
    {
      "id": 1,
      "category": "core",
      "prompt": "Review this pull request diff: [paste diff]",
      "expected_output": "Security check → logic review → style → summary with go/no-go",
      "assertions": [
        { "type": "all_steps_covered", "steps": ["security", "logic", "style", "summary"] },
        { "type": "step_order", "before": "security", "after": "summary" },
        { "type": "quality", "rubric": "Does the review give a clear go/no-go decision?" }
      ]
    }
  ]
}
```

---

### Type Detection at `init`

The 4th SOP question (`Is this a code-generation skill?`) maps to type:

```
Code generation? yes → skill_type: "coding"  → prompts for code_language, linter path
Code generation? no  → skill_type: "workflow" → prompts for required workflow steps
```

User can override `skill_type` manually in `evals.json`.

---

## 7b. LLM-Assisted Eval Generation (`eval generate`)

For users who don't know what test cases to write:

```
$ skill-arena eval generate flink-skill

Reading SKILL.md...
Calling claude-sonnet-4-6 to suggest eval cases...

Suggested eval cases:
  [1] core      — "Write a Flink job reading from Kafka with 5-min tumbling window"
  [2] core      — "Implement a Flink DataStream job with exactly-once semantics"
  [3] edge      — "Handle late-arriving events with a 30-second allowed lateness"
  [4] error_diagnosis — "Checkpoint timeout: 'Checkpoint 123 expired before completing'"

? Which to add? (space to select, enter to confirm)
> [x] 1  [x] 4  [ ] 2  [ ] 3

✓ 2 cases added to evals.json
⚠ Still need: 1 edge case before eval run is unblocked
```

The LLM is given the full `SKILL.md` + the skill type + instructions to produce prompts that stress-test the description's claimed capabilities.

Generated cases are **suggestions only** — user reviews and approves each one before they're saved. The tool never auto-writes evals silently.

---

## 8. Eval Runner: Isolation Model

**Key design insight:** When testing one skill, other skills in the project must NOT be injected. The eval runs in isolation — only the skill under test is in context.

```
skill-arena eval run flink-skill
        │
        ├─── Request A (WITH skill)
        │    system = base prompt + ONLY flink-skill/SKILL.md
        │    user   = eval.prompt
        │    → response_A
        │
        └─── Request B (WITHOUT skill)
             system = base prompt only  ← no skills at all
             user   = eval.prompt
             → response_B

Both run in parallel.
Diff: response_A vs response_B (unified diff format)
Assertions evaluated on each response independently.
```

This isolation is why project-level (`.claude/skills/`) is the right scope — you're testing one skill at a time, not the full environment.

---

## 8. SKILL.md Validation Rules

| Rule | Severity |
|------|----------|
| Frontmatter `name` present | Error |
| Frontmatter `description` present, ≥ 50 chars | Error |
| Description contains at least one trigger keyword/phrase | Warning |
| Body ≤ 500 lines | Warning at 400, Error at 500 |
| Has `## Output Format` section | Warning |
| Has at least one file in `references/` if body references external docs | Warning |

---

## 9. Configuration (`config`)

Stored in `~/.skill-arena/config.json` (user-level, not committed to git):

```json
{
  "api_base_url": "https://api.anthropic.com",
  "api_key": "sk-ant-...",
  "default_model": "claude-sonnet-4-6"
}
```

Set via interactive prompts:

```
$ skill-arena config

? API base URL  [https://api.anthropic.com]
> https://api.openai.com/v1

? API key
> sk-...

? Default model
> gpt-4o
```

Supports any OpenAI-compatible endpoint + Anthropic. User-provided model string is sent as-is — no model validation.

---

## 10. Assertion Types (Full Reference)

Common assertions (available for all skill types):

| Type | Behavior |
|------|----------|
| `contains` | Response text includes the specified string |
| `not_contains` | Response text does NOT include the specified string |
| `quality` | LLM judge: sends response + rubric to model, returns pass/fail + reason. Costs one extra API call per assertion. |

Coding skill assertions — see §7 Type A.

Workflow skill assertions — see §7 Type B.

---

## 11. Architecture (Go CLI)

```
skill-arena/
├── cmd/
│   ├── main.go
│   ├── init.go          # skill-arena init
│   ├── validate.go      # skill-arena validate
│   ├── eval.go          # skill-arena eval run / add / history
│   └── config.go        # skill-arena config
├── internal/
│   ├── skill/
│   │   ├── scaffold.go  # Generate SKILL.md from 4 questions
│   │   ├── validate.go  # SOP compliance checks
│   │   └── io.go        # Read/write SKILL.md, evals.json
│   ├── eval/
│   │   ├── runner.go    # Parallel API calls, diff generation
│   │   ├── assert.go    # Assertion evaluation
│   │   └── history.go   # Read/write run history
│   └── llm/
│       └── client.go    # OpenAI-compatible HTTP client (works with Anthropic too)
├── go.mod
└── README.md
```

**Distribution:** Single binary. Build with `go build`, release via GitHub Releases. No runtime, no package manager.

---

## 12. Tech Stack

| Concern | Choice | Reason |
|---------|--------|--------|
| Language | Go | Single binary, no runtime, fast, good CLI libs |
| CLI framework | `cobra` | Standard Go CLI, flag parsing, help generation |
| Interactive prompts | `bubbletea` / `survey` | Good TUI input for init/config flows |
| HTTP client | stdlib `net/http` | No SDK needed, works with any OpenAI-compatible endpoint |
| Diff | `go-diff` (Dmitri Shuralyov) | Unified diff output, standard format |
| Config storage | `~/.skill-arena/config.json` | Plain JSON, stdlib only |
| Eval history | `.skill-arena/history/<skill>/<timestamp>/` | Plain files, no DB |

---

## 13. Out of Scope (v1)

- Web UI / browser interface
- Publishing or sharing skills to a registry
- Multi-user / team features
- Skill versioning or git integration
- Auto-rewriting `SKILL.md` based on eval results (tool measures, human decides)
- User-level skill testing (`~/.claude/skills/`) — project-level only
- Installing linters/compilers — user must provide the binary path in config

---

## 14. Success Criteria (MVP)

- [ ] Single binary, no install dependencies
- [ ] `skill-arena init` → first eval run in < 10 minutes
- [ ] With/without comparison shows measurable improvement for a well-written skill
- [ ] Works with Anthropic API and any OpenAI-compatible endpoint
- [ ] Config and history stored locally, nothing sent anywhere except LLM API calls

---

## 15. Resolved Decisions

| Question | Decision |
|----------|---------|
| API key source | UI paste via `skill-arena config` (stored in `~/.skill-arena/config.json`) |
| Skill scope | Project-level only (`.claude/skills/`) — isolation by design |
| Model support | BYOM: user sets API base URL + API key + model name |
| Distribution | Go binary via GitHub Releases — no npm, no Node.js |
| Diff format | Unified diff written to file, viewable with `$DIFFTOOL` or `delta` |

---

## 16. One Remaining Open Question

**How to display long diffs in terminal?**

Option A — Print inline (truncated to 50 lines, full diff saved to file)
Option B — Open in `$DIFFTOOL` or `$PAGER` automatically
Option C — Write a markdown report to `.skill-arena/history/.../report.md` and print the path

Leaning toward **Option A + C**: print summary inline, save full report as markdown. User opens it in their editor.
