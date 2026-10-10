# Contributing to tokenix

Thanks for helping make tokenix better! Bug reports, ideas, docs fixes, and new model profiles are all welcome.

## Getting started

```bash
git clone https://github.com/hasirciogluhq/tokenix
cd tokenix
go build ./...
go test ./...
go run . --data "hello world"
```

Requires Go 1.22 or newer.

## Workflow

1. Fork the repo and branch from `development`.
2. Make your change, with tests where it makes sense.
3. Run `gofmt -l .`, `go vet ./...` and `go test ./...`.
4. Open a pull request against `development`.

Releases are cut by maintainers: `development` is merged into `main`, then a `vX.Y.Z` tag triggers GoReleaser, which publishes binaries and updates the Homebrew formula.

## Provider profiles

Models and prices come from the live catalog (`internal/models`). Token ratios live in `internal/token/estimator.go` as per-provider profiles; providers without one use the global profile. To add a provider, allow it in `providers` in `internal/models/models.go` and, optionally, add a profile.

## Style

- Keep the CLI output stable; JSON field names are part of the public interface.
- User-facing text (flags, errors, docs) is in English.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages (`feat:`, `fix:`, `docs:` ...).
