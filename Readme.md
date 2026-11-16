# TermAI

**The AI workspace for your terminal.**

TermAI is a local-first, provider-agnostic AI command-line application
for developers. It brings conversational AI into the terminal while
keeping the interface lightweight, the data local by default, and the
architecture open to multiple model providers.

> **Project status:** Early development. The current implementation is a
> functional Ollama-backed CLI with interactive chat, one-shot chat,
> model listing, YAML configuration, and SQLite-backed conversation
> history. APIs, commands, and configuration may change before v1.0.

## What works today

-   Interactive terminal chat with streamed responses.
-   One-shot chat through the CLI.
-   Ollama integration for local model inference.
-   List locally available Ollama models.
-   Select a model through configuration or a chat flag.
-   YAML configuration with defaults, strict field validation, and a
    custom config path.
-   SQLite persistence for chat sessions and messages.
-   Resume, inspect, clear, and delete conversation sessions from the
    REPL.
-   Atomic persistence of each user/assistant exchange.
-   Graceful handling of Ctrl+C and provider errors.
-   Structured logging with an optional verbose/debug mode.
-   Automated tests for CLI commands, configuration, Ollama HTTP
    behavior, storage, and REPL behavior.

The list above describes the implemented project state, not the full
long-term product vision. See the private product strategy document
maintained by the project owner for broader direction.

## Current architecture

``` text
.
├── main.go
├── commands.go
├── main_test.go
└── internal/
    ├── config/          # YAML configuration and validation
    ├── conversation/    # Session and message domain models
    ├── llm/             # Provider-neutral request/response interfaces
    ├── provider/
    │   └── ollama/      # Ollama HTTP client and streaming
    └── storage/
        └── sqlite/      # SQLite repositories and migrations
```

The application is organized around a provider-neutral LLM interface.
The current provider implementation is Ollama. SQLite is used for local
conversation persistence, with migrations managed by Goose and a pure-Go
SQLite driver.

## Requirements

-   Go version compatible with the version declared in `go.mod`.
-   [Ollama](https://ollama.com/) installed and running.
-   At least one model pulled into Ollama.

For example:

``` bash
ollama pull llama3.2:3b
```

## Run from source

Clone the repository and enter its directory:

``` bash
git clone https://github.com/Rugved7/termai.git
cd termai
```

Download dependencies and run the CLI:

``` bash
go mod download
go run . --help
```

To build a local binary:

``` bash
go build -o termai .
./termai --help
```

The repository URL above is the intended public project URL; update it
if the repository is published under a different owner or name.

## Quick start

Make sure Ollama is running and a model is available:

``` bash
ollama list
```

Start an interactive session:

``` bash
./termai chat
```

Use a different model for a chat session:

``` bash
./termai chat -m llama3.2:3b
```

List models available from the configured Ollama server:

``` bash
./termai models
```

Show the available commands and flags:

``` bash
./termai --help
./termai chat --help
```

## Configuration

TermAI loads YAML configuration from:

``` text
~/.config/termai/config.yaml
```

If the file does not exist, the application uses built-in defaults.
Create the default configuration file with:

``` bash
./termai config init
```

Inspect the active configuration and its path:

``` bash
./termai config show
./termai config path
```

Current default configuration:

``` yaml
provider: ollama
model: llama3.2:3b
data_dir: ~/.termai
ollama:
  base_url: http://localhost:11434
```

The actual generated configuration expands the data directory to the
user's home directory. The configuration loader rejects unknown fields
and malformed or ambiguous YAML rather than silently accepting typos.

Use a non-default configuration file with the root flag:

``` bash
./termai --config /path/to/config.yaml config show
./termai --config /path/to/config.yaml chat
```

Enable verbose diagnostic logging with:

``` bash
./termai --verbose models
```

Verbose logging is intended for operational diagnostics. Prompt and
response contents should not be written to application logs.

## Conversation data

Conversation data is stored locally in SQLite under the configured
`data_dir`. By default, this is:

``` text
~/.termai/termai.db
```

The interactive chat currently supports these slash commands:

  Command           Purpose
  ----------------- ----------------------------------------------------------
  `/help`           Display REPL commands, if available in the current build
  `/exit`           Exit the interactive session
  `/clear`          Start a new conversation session
  `/models`         List available Ollama models
  `/model`          Display or change the active model
  `/model <name>`   Switch the active model
  `/history`        List saved sessions
  `/resume <id>`    Resume a saved session
  `/delete <id>`    Delete a saved session

Conversation history is persisted locally. Keep in mind that deleting
the TermAI data directory removes the local database and its stored
conversations.

## Development

Run the test suite:

``` bash
go test ./...
```

Run tests with the race detector:

``` bash
go test -race ./...
```

Run static analysis:

``` bash
go vet ./...
```

Format Go source:

``` bash
gofmt -w .
```

Before opening a pull request, run all four checks and ensure they pass.

## Roadmap

The project is being developed in stages. Near-term work is focused on
making the current CLI reliable and ready for public use:

-   Complete integration and edge-case coverage.
-   Improve documentation and onboarding.
-   Add continuous integration.
-   Produce reproducible cross-platform builds.
-   Establish versioning and release automation.
-   Define a stable v1.0 scope and compatibility expectations.

Longer-term product ideas include additional model providers, richer
project context, reusable prompts and workflows, and optional
team-oriented capabilities. These are exploration areas, not commitments
or features currently available.

## Contributing

TermAI is intended to become an open-source project. Until contribution
guidelines and the v1.0 compatibility policy are published, please open
an issue to discuss substantial changes before investing in a large pull
request.

When contributing:

1.  Keep provider-specific behavior behind the provider interface.
2.  Preserve local-first defaults.
3.  Avoid logging prompts, responses, credentials, or other private user
    content.
4.  Add tests for new behavior and regressions.
5.  Keep the CLI and configuration behavior documented.

## License

The project license has not yet been finalized. A license file will be
added before the first public release. Until then, do not assume that
the repository grants permission to use, modify, or redistribute the
code.