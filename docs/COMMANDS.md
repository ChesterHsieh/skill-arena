# CLI Command Reference

<!-- AUTO-GENERATED from cobra command definitions -->

## `skill-arena init <name>`

Scaffold a new skill interactively.

Asks 4 questions from the Agent Skill SOP, then:
1. Calls the configured LLM (if API key is set) to expand raw answers into a structured SKILL.md using the SOP principles as a system prompt
2. Falls back to a static template if no API key is configured
3. Creates `evals.json` with 3 template eval cases for the skill type

```
.claude/skills/<name>/
├── SKILL.md        ← generated + LLM-structured
├── evals.json      ← template with 3 empty cases
└── references/     ← empty, add supporting docs here
```

---

## `skill-arena validate <name>`

Check `SKILL.md` against SOP compliance rules. Exits with code `1` if any errors are found.

| Rule | Severity |
|------|----------|
| Frontmatter `name` present | Error |
| Frontmatter `description` ≥ 50 chars | Error |
| Description contains trigger keywords | Warning |
| Body ≤ 500 lines (warn at 400) | Warning / Error |
| `## Output Format` section present | Warning |
| `references/` has files (if body references it) | Warning |

---

## `skill-arena edit <name>`

Open `SKILL.md` in `$EDITOR`. Falls back to `nano`, then `vim` if `$EDITOR` is not set.

---

## `skill-arena config`

Interactive prompts to set API configuration. Stored at `~/.skill-arena/config.json`.

See [CONFIG.md](./CONFIG.md) for all fields and provider examples.

---

## `skill-arena eval add <name>`

Add an eval case interactively. Prompts for:
- Category (`core` / `edge` / `error_diagnosis` for coding; `core` / `partial` / `edge` for workflow)
- Prompt text
- Expected output description
- Assertion loop (add assertions one by one)

Appends to `evals.json`.

---

## `skill-arena eval generate <name>`

Use the LLM to suggest eval cases from the skill's `SKILL.md`. Requires API key.

1. Reads `SKILL.md`
2. Calls the LLM with instructions to generate 5 diverse cases matching the skill type's required categories
3. Presents suggestions — user picks which to keep
4. Appends approved cases to `evals.json` (no silent writes)

---

## `skill-arena eval run <name>`

Run the with/without comparison eval. **Requires API key.**

**Gate:** blocked until `evals.json` has ≥ 3 cases covering all required categories:
- Coding: `core`, `edge`, `error_diagnosis`
- Workflow: `core`, `partial`, `edge`

**Isolation:** only the skill under test is injected — no other skills from `.claude/skills/` are included.

**Concurrency:** max 3 eval cases run in parallel (6 total API connections) to stay within rate limits.

**Output (A+C):**
- Inline summary printed to terminal (truncated diff, per-case assertion scores)
- Full markdown report saved to `.skill-arena/history/<name>/<timestamp>/report.md`

---

## `skill-arena eval history <name>`

Show a table of past eval runs with WITH/WITHOUT pass rates, impact delta, and trend indicator.

<!-- END AUTO-GENERATED -->

## File locations

| Path | Purpose |
|------|---------|
| `.claude/skills/<name>/SKILL.md` | Skill definition |
| `.claude/skills/<name>/evals.json` | Eval cases + assertions |
| `.claude/skills/<name>/references/` | Supporting reference docs |
| `~/.skill-arena/config.json` | API config (user-level) |
| `.skill-arena/history/<name>/<ts>/report.md` | Full eval report |
| `.skill-arena/history/<name>/<ts>/results.json` | Machine-readable results |
