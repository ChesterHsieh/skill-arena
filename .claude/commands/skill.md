Run skill-arena CLI commands from within Claude Code.

Usage examples:
- `/skill init my-skill` — scaffold a new skill
- `/skill validate my-skill` — check SOP compliance
- `/skill eval run my-skill` — run with/without eval
- `/skill eval generate my-skill` — generate eval cases with LLM
- `/skill eval history my-skill` — show past runs

## Instructions

The argument(s) passed after `/skill` map directly to skill-arena subcommands.

1. Check that `skill-arena` is installed:
   ```bash
   which skill-arena
   ```
   If not found, offer to install it:
   ```bash
   curl -sSL https://raw.githubusercontent.com/ChesterHsieh/skill-arena/main/install.sh | sh
   ```

2. Run the command from the project root (current working directory):
   ```bash
   skill-arena $ARGUMENTS
   ```
   Where `$ARGUMENTS` is everything the user typed after `/skill`.

3. Show the full output inline. If the command writes a report file, read and display the report.

4. For interactive commands (`init`, `eval add`, `config`) that require prompts: instead of running them in non-interactive mode, ask the user for each required input field yourself, then construct the files directly without calling the CLI interactively. Use the skill file format from the project's RFP.md.
