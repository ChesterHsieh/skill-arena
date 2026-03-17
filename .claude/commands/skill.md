Run skill-arena workflows natively inside Claude Code — no binary required.

Usage:
- `/skill init <name>` — scaffold a new skill interactively
- `/skill validate <name>` — check SOP compliance
- `/skill eval run <name>` — run with/without comparison
- `/skill eval add <name>` — add an eval case
- `/skill eval history <name>` — show past runs

---

## How to handle each command

Claude handles all commands natively by reading/writing files directly. The `skill-arena` binary is NOT required.

---

### `init <name>`

Ask the user these 4 questions one by one:
1. What should the AI do with this skill?
2. When should it trigger? (list keywords or user intent)
3. What is the expected output format?
4. Is this a code-generation skill? (yes → skill_type: "coding", no → skill_type: "workflow")

Then create these files:

**`.claude/skills/<name>/SKILL.md`**
```
---
name: <name>
description: >
  <answer to Q1>. Trigger when user mentions: <answer to Q2>.
---

# <Name> Skill

## When to Activate
Activate when user mentions: <Q2 keywords>.

## Workflow
1. [Step 1]
2. [Step 2]
3. [Step 3]

## Output Format
<answer to Q3>

## Notes
- [Add caveats here]
```

**`.claude/skills/<name>/evals.json`**
For coding skills — template with categories: core, edge, error_diagnosis.
For workflow skills — template with categories: core, partial, edge.

```json
{
  "skill_name": "<name>",
  "skill_type": "<coding|workflow>",
  "evals": [
    {"id": 1, "category": "core", "prompt": "", "expected_output": "", "assertions": []},
    {"id": 2, "category": "<edge|partial>", "prompt": "", "expected_output": "", "assertions": []},
    {"id": 3, "category": "<error_diagnosis|edge>", "prompt": "", "expected_output": "", "assertions": []}
  ]
}
```

Also create an empty `.claude/skills/<name>/references/` directory.

---

### `validate <name>`

Read `.claude/skills/<name>/SKILL.md` and check these rules. Print ✓, ⚠, or ✗ for each:

| Rule | Severity |
|------|----------|
| Frontmatter `name` field present | Error |
| Frontmatter `description` ≥ 50 chars | Error |
| Description contains trigger keywords | Warning |
| Body ≤ 500 lines (warn at 400) | Warning/Error |
| `## Output Format` section present | Warning |
| `references/` directory exists with ≥ 1 file | Warning |

Print summary: "N warnings, N errors" or "All checks passed."

---

### `eval run <name>`

Read `.claude/skills/<name>/SKILL.md` and `.claude/skills/<name>/evals.json`.

**Gate check:** verify evals.json has cases covering all required categories:
- coding: core, edge, error_diagnosis
- workflow: core, partial, edge

If missing, print which categories are absent and stop.

For each eval case, run a with/without comparison **using your own inference**:

**WITH skill:** Imagine you have the full SKILL.md injected into your context. Answer the eval prompt as if the skill is active.

**WITHOUT skill:** Answer the same eval prompt with no skill context — just a generic helpful assistant.

Then for each case:
1. Show both responses side by side (truncated to ~20 lines each)
2. Evaluate each assertion against both responses
3. Show pass/fail per assertion

Print summary:
```
Summary  WITH:    X/Y assertions passed (N%)
         WITHOUT: X/Y assertions passed (N%)
         Impact:  +Npp assertion improvement
```

Save a markdown report to `.skill-arena/history/<name>/<timestamp>/report.md` with the full responses and assertion results.

---

### `eval add <name>`

Read `.claude/skills/<name>/evals.json` to see existing cases.

Ask the user:
1. Category (core / edge / error_diagnosis / partial)
2. Prompt text
3. Expected output description
4. Add assertions? (loop: type → fields → continue?)

Append the new case to evals.json with the next available ID.

---

### `eval history <name>`

List all directories under `.skill-arena/history/<name>/`. For each, read `report.md` and extract the summary line. Print as a table:

```
Timestamp              WITH%   WITHOUT%  Impact
2026-03-16 10:00       78%     22%       +56pp  [latest]
2026-03-15 14:32       56%     22%       +34pp
```

If no history exists, print: "No eval runs yet. Run: /skill eval run <name>"
