# Ceerat ChatGPT Client

Go-backed version of the legacy `ai/chatgptclient` UI.

The app keeps the chat-style browser experience and replaces the old Node/Assistants API server with a small Go HTTP server using OpenAI chat completions streaming.

## Run

```bash
export OPENAI_API_KEY="sk-..."
go run ./ai/ceerat-chatgpt-client
```

Default URL:

```text
http://localhost:3010
```

Environment:

```text
CEERAT_CHATGPT_CLIENT_PORT=3010
OPENAI_MODEL=gpt-4.1-mini
OPENAI_API_KEY=...
```
