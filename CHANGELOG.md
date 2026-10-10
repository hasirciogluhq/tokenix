# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/) and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.1.2] - 2026-10-10

### Changed
- The default (global) estimate now includes pricing: the average input price across all known models and the resulting cost. It loads the model catalog (cached 24h), falling back to the built-in list when offline.

## [1.1.1] - 2026-10-10

### Changed
- **Breaking:** the hard-coded model list is gone. Without flags tokenix prints a single global estimate; use `--model` (single or comma-separated) or `--all` for model-specific results.
- Text output is now a table; JSON schema changed (`counts`, `results[]`, `summary`, `catalog`).
- Documentation, error messages, and tests are now fully in English.

### Added
- Live model catalog with input pricing (OpenAI, Anthropic, Gemini, Mistral, xAI, DeepSeek), cached for 24 hours in the user cache dir.
- `--all`, `--refresh`, `tokenix models [filter]`, `tokenix cache path|clear`.
- Cost estimates per model.
- `LICENSE` (MIT), `CONTRIBUTING.md`, issue/PR templates, and a GitHub Pages site under `docs/`.

## [1.1.0] - 2026-10-10

### Added
- `--dir` for recursive directory scanning.
- `--include` / `--exclude` glob filters; JSON output lists the counted files.

## [1.0.0] - 2026-04-16

### Added
- Initial release: `--data`, `--file`, stdin input, multi-model estimates, JSON output, Homebrew and install-script distribution.
