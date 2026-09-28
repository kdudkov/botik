# Repository Guidelines

## Project Structure & Module Organization

Botik is a Go Telegram bot integrating MQTT, Frigate, home automation, and alert notifications. The module is `botik`.

- `cmd/botik/`: application entry point, configuration, HTTP server, and MQTT handling.
- `cmd/botik/answer/`: command matching and response handlers; adjacent `*_test.go` files cover behavior.
- `cmd/botik/alert/`: alert lifecycle and notifications; `template/` contains embedded HTML message templates.
- `internal/api/`: Mahno and InfluxDB clients; `internal/util/`: shared utilities.
- `cmd/influx_test/`: manual InfluxDB query executable with a hard-coded endpoint, not an automated test suite.
- `dist/`: generated binaries; `botik.yml.example`: configuration example.

## Build, Test, and Development Commands

Use Go 1.26.0 or newer and Task to run `Taskfile.yml` targets.

- `task build`: tidy dependencies, clean `dist/`, and build both executables.
- `task test`: tidy dependencies and run `go test ./...`.
- `go test ./cmd/botik/answer -run TestWords`: run one focused test.
- `task format`: run `gofmt` and `goimports`, grouping local `botik` imports.
- `task lint`: run separately installed `golangci-lint`.
- `task precommit` (also `task`): clean, tidy, format, test, and lint new changes. Its lint comparison depends on the `master` branch and Git history.
- `go run ./cmd/botik`: start the bot using `botik.yml` in the working directory.

`task tools` installs `goimports`. Review dependency and formatting changes produced by Task commands.

## Coding Style & Naming Conventions

Use standard Go formatting with tabs. Keep package names lowercase, exported identifiers in PascalCase, and unexported identifiers in camelCase. Match neighboring files and keep changes focused. Use the existing `log/slog` logging patterns and check errors from external operations.

## Testing Guidelines

Tests use Go's `testing` package and `testify/assert`. Place tests beside implementation files as `*_test.go`, with functions named `TestBehavior`. Use mocks for external APIs and add focused regression tests for behavior changes. No numeric coverage threshold is configured.

## Commit & Pull Request Guidelines

History uses short subjects such as `fix users` and `versions`, without a consistent prefix convention. Write concise, descriptive subjects. PRs should explain the behavior changed, link relevant issues, and report validation commands and results, including any checks not run.

## Configuration & Agent Instructions

Copy `botik.yml.example` to ignored `botik.yml`; replace example credentials and endpoints before running. Use a development bot because startup changes Telegram webhook state. Never commit tokens or private configuration.

For code work, use `ponytail` and `karpathy-guidelines`. Use `faster-search` with `rg` or `fd`, never `grep` or `find`. Ask for clarification when requirements are uncertain.
