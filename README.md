# Dewclaw

Dewclaw is a Go library and set of CLI tools built on top of the Google Gemini AI.

## Installation

```bash
go install github.com/rspier/dewclaw/cmd/dewclaw@latest
go install github.com/rspier/dewclaw/cmd/quickdesc@latest
```

## Configuration

Both tools require a Gemini API key. You can provide it in two ways:

1.  **Environment Variable**: Set `GEMINI_API_KEY`.
2.  **Config File**: Place the raw API key in `~/.config/dewclaw/api_key`.

## Tools

### `quickdesc`

Automatically extracts a diff from your current Git or JJ (Jujutsu) repository and generates a concise, conventional commit message.

**Usage:**

```bash
quickdesc [-v] [-U context_lines]
```

### `dewclaw` (Generic CLI)

A generic interface to the Gemini content generation engine.

**Usage:**

```bash
dewclaw -s "System prompt" -t "User prompt/text" [-v]
```

## Library Usage

You can use the `dewclaw` package in your own Go projects:

```go
import "github.com/rspier/dewclaw"

func main() {
    apiKey, _ := dewclaw.GetAPIKey()
    client := dewclaw.NewClient(apiKey)

    response, usage, err := client.GenerateContent("You are a helpful assistant", "Explain quantum physics in one sentence.")
    // ...
}
```
