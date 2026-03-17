# Configuration Reference

Config file: `~/.skill-arena/config.json`
Set interactively: `skill-arena config`

<!-- AUTO-GENERATED from internal/config/config.go -->

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `api_base_url` | No | `https://api.anthropic.com` | LLM API base URL. Any OpenAI-compatible endpoint works. |
| `api_key` | Yes | — | API key for the configured endpoint. Stored with `0600` permissions. |
| `default_model` | No | `claude-sonnet-4-6` | Model name sent as-is to the API. Not validated — use any model your API supports. |
| `linter_path` | No | — | Path to a linter binary for `code_style` assertions (e.g. `eslint`, `ruff`, `golangci-lint`). |

<!-- END AUTO-GENERATED -->

## Provider examples

**Anthropic (default)**
```json
{
  "api_base_url": "https://api.anthropic.com",
  "api_key": "sk-ant-...",
  "default_model": "claude-sonnet-4-6"
}
```

**OpenAI**
```json
{
  "api_base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "default_model": "gpt-4o"
}
```

**Local (Ollama)**
```json
{
  "api_base_url": "http://localhost:11434/v1",
  "api_key": "ollama",
  "default_model": "llama3.2"
}
```

## Provider detection

The client auto-detects the API format from `api_base_url`:

- Contains `anthropic.com` → Anthropic Messages API (`POST /v1/messages`)
- Anything else → OpenAI Chat Completions (`POST /v1/chat/completions`)

## Security

- Config file is written with `0600` permissions (owner read/write only)
- Config directory (`~/.skill-arena/`) is created with `0700`
- Never commit `~/.skill-arena/config.json` — it contains your API key
