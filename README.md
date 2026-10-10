# tokenix

[![CI](https://github.com/hasirciogluhq/tokenix/actions/workflows/ci.yml/badge.svg)](https://github.com/hasirciogluhq/tokenix/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/hasirciogluhq/tokenix)](https://github.com/hasirciogluhq/tokenix/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A tiny CLI that estimates how many tokens your prompt, file, or whole project will cost across popular LLMs — offline, instantly, with no API keys.

```console
$ tokenix --dir . --include '*.go' --exclude '*_test.go' --model gpt-4o,claude-sonnet-4-5
source: dir:. (7 files)
characters: 27206
words: 3537

MODEL              MIN   MAX   AVG   NET   IN $/1M  COST (NET)
gpt-4o             5553  8002  6778  6974  $2.5000  $0.0174
claude-sonnet-4-5  5915  9069  7492  7776  $3.0000  $0.0233

overall: min 5553, max 9069, avg 7135, net 7375, spread 3516
```

## Features

- Estimate from a string (`--data`), a file (`--file`), stdin, or a whole directory tree (`--dir`)
- Recursive directory scanning with `--include` / `--exclude` glob filters
- **Global mode by default**: one model-agnostic estimate, no network needed
- **Model mode**: `--model gpt-4o` or `--model gpt-4o,claude-sonnet-4-5`, or `--all` for every known model
- **Input pricing** per model (USD per 1M tokens and estimated cost)
- Model list and prices come from a live catalog and are cached locally for 24 hours; `--refresh` forces a re-download
- Text table or JSON output (JSON also lists the files that were counted)
- Single static binary, no API keys

> **Note:** tokenix gives *estimates* based on character and word heuristics, not exact tokenizer output. Prices come from a community catalog. Use it for budgeting and sanity checks, not billing.

## Models and pricing

Without `--model` / `--all`, tokenix prints a single `global` estimate and never touches the network.

When you select models, tokenix loads a catalog of first-party chat models (OpenAI, Anthropic, Google Gemini, Mistral, xAI, DeepSeek) from the [LiteLLM price catalog](https://github.com/BerriAI/litellm/blob/main/model_prices_and_context_window.json):

| Step | Behavior |
| --- | --- |
| Fresh cache (< 24h) | Used as is, no network |
| Stale or missing cache | Downloaded, then cached |
| `--refresh` | Cache ignored, always downloaded |
| Download fails | Falls back to the stale cache, then to a small built-in list, with a warning |

The cache lives in your user cache directory (`tokenix cache path` prints the exact file; `~/Library/Caches/tokenix` on macOS, `~/.cache/tokenix` on Linux).

```bash
tokenix models              # list every known model with prices
tokenix models claude       # filter by id or provider
tokenix cache path          # where the cache file lives
tokenix cache clear         # delete the cache
```

Model names are case-insensitive and accept any unambiguous prefix (`gemini-2.5-pr` resolves to `gemini-2.5-pro`). Unknown names get suggestions.

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

# One model, several models, or all of them (with pricing)
tokenix --data "test" --model gpt-4o
tokenix --data "test" --model gpt-4o,claude-sonnet-4-5
tokenix --data "test" --all

# Force a fresh model/pricing download, JSON output
tokenix --data "test" --model gpt-4o --refresh --format json
```

### Flags

| Flag | Short | Description |
| --- | --- | --- |
| `--data` | `-d` | Raw text to estimate |
| `--file` | `-f` | Path to a file |
| `--dir` | `-D` | Directory to scan recursively |
| `--include` | `-i` | Only count files matching these globs (repeatable or comma-separated) |
| `--exclude` | `-x` | Skip files/directories matching these globs (repeatable or comma-separated) |
| `--model` | `-m` | Model id(s), comma-separated or repeated. Default: none (global estimate) |
| `--all` | `-a` | Estimate for every known model |
| `--refresh` | | Ignore the cache and re-download the model catalog |
| `--format` | | `text` (default) or `json` |
| `--version` | | Print version |

Input precedence: `--dir`, then `--data`, then `--file`, then piped stdin.

### Filter rules

- A pattern matches if it matches the relative path, the file name, or the name of any parent directory (`*.go`, `vendor`).
- `dir/**` matches everything below `dir`.
- An empty `--include` means "all files"; `--exclude` always wins.
- `.git`, `node_modules`, and binary files are skipped automatically.

## How estimation works

Each provider family has a profile with a min and max characters-per-token ratio and a bias; models from providers without a dedicated profile use the global profile. The estimate combines a character-based range with a word-based range, then derives `average` and a `net_estimate` that leans toward the upper bound by the profile's bias. Cost is `tokens × input price`. With several models, an overall summary aggregates them.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Development happens on `development`; releases are cut from `main` by pushing a `vX.Y.Z` tag.

## License

[MIT](LICENSE)
