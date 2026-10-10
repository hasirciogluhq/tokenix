# tokenix

[![CI](https://github.com/hasirciogluhq/tokenix/actions/workflows/ci.yml/badge.svg)](https://github.com/hasirciogluhq/tokenix/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/hasirciogluhq/tokenix)](https://github.com/hasirciogluhq/tokenix/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A tiny CLI that estimates how many tokens your prompt, file, or whole project will cost across popular LLMs — offline, instantly, with no API keys.

```console
$ tokenix --dir . --include '*.go' --exclude '*_test.go' --model gpt-4o
source: dir:. (5 files)
characters: 13724
words: 1641

model: gpt-4o
  min_tokens: 2801
  max_tokens: 4037
  average_tokens: 3419
  net_estimate: 3604
```

## Features

- Estimate from a string (`--data`), a file (`--file`), stdin, or a whole directory tree (`--dir`)
- Recursive directory scanning with `--include` / `--exclude` glob filters
- Per-model min / max / average / net estimates plus an overall summary with spread
- Text or JSON output (JSON also lists the files that were counted)
- Single static binary, no network access, no API keys

> **Note:** tokenix gives *estimates* based on character and word heuristics, not exact tokenizer output. Use it for budgeting and sanity checks, not billing.

## Supported models

| Key | Family |
| --- | --- |
| `gpt-4o-mini` | OpenAI |
| `gpt-4o` | OpenAI |
| `claude-3.5-sonnet` | Anthropic |
| `claude-3-haiku` | Anthropic |
| `gemini-1.5-flash` | Google |
| `gemini-1.5-pro` | Google |

## Install

### Homebrew

```bash
brew tap hasirciogluhq/tap
brew install tokenix
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/hasirciogluhq/tokenix/main/scripts/install.sh | bash
# or pin a version
curl -fsSL https://raw.githubusercontent.com/hasirciogluhq/tokenix/main/scripts/install.sh | bash -s -- v1.1.0
```

### Go

```bash
go install github.com/hasirciogluhq/tokenix@latest
```

### Prebuilt binaries

Download archives for macOS, Linux, and Windows (amd64/arm64) from the [releases page](https://github.com/hasirciogluhq/tokenix/releases).

## Usage

```bash
# Raw text
tokenix --data "Hello, world"

# A file
tokenix --file ./prompt.txt

# stdin
cat prompt.txt | tokenix --model gpt-4o

# A directory, recursively, with filters
tokenix --dir ./src
tokenix --dir . --include '*.go' --exclude '*_test.go,vendor'

# Several models, JSON output
tokenix --data "test" --model gpt-4o,claude-3.5-sonnet --format json
```

### Flags

| Flag | Short | Description |
| --- | --- | --- |
| `--data` | `-d` | Raw text to estimate |
| `--file` | `-f` | Path to a file |
| `--dir` | `-D` | Directory to scan recursively |
| `--include` | `-i` | Only count files matching these globs (repeatable or comma-separated) |
| `--exclude` | `-x` | Skip files/directories matching these globs (repeatable or comma-separated) |
| `--model` | `-m` | Model key, comma-separated list, or `all` (default) |
| `--format` | | `text` (default) or `json` |
| `--version` | | Print version |

Input precedence: `--dir`, then `--data`, then `--file`, then piped stdin.

### Filter rules

- A pattern matches if it matches the relative path, the file name, or the name of any parent directory (`*.go`, `vendor`).
- `dir/**` matches everything below `dir`.
- An empty `--include` means "all files"; `--exclude` always wins.
- `.git`, `node_modules`, and binary files are skipped automatically.

## How estimation works

Each model has a profile with a min and max characters-per-token ratio and a bias. The estimate combines a character-based range with a word-based range, then derives `average` and a `net_estimate` that leans toward the upper bound by the model's bias. The overall summary aggregates all selected models.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Development happens on `development`; releases are cut from `main` by pushing a `vX.Y.Z` tag.

## License

[MIT](LICENSE)
