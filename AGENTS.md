# Development Guide for Max Bot

## Build, Lint, and Test Commands

### Build Commands
- `go build -o bin/max-bot ./main.go` - Compiles the bot binary into the `bin` directory.
- `docker build -t max-bot .` - Builds the Docker image for the bot.
- `go run main.go` - Runs the bot directly from source (useful for development).

### Linting
- `golangci-lint run ./...` - Runs all linters configured in the project.
- `go vet ./...` - Runs Go's built‑in static analysis tool.
- `staticcheck ./...` - Executes the staticcheck analyzer for additional checks.
- `goimports -w .` - Formats imports and removes unused ones across the codebase.

### Testing
- `go test ./...` - Executes all unit tests in the repository.
- `go test ./handlers -run TestHandleMessage -v` - Runs a single test file with verbose output.
- `go test ./handlers -count=1` - Executes a test only once, bypassing any cached results.
- `go test -cover ./...` - Shows test coverage for all packages.

### Example Single‑Test Execution
```bash
go test ./handlers -run TestHandleMessage -v -count=1
```

## Code Style Guidelines

### File Organization
- Keep each package in its own directory (`handlers`, `bot`, `utils`).
- Place implementation files (`*.go`) alongside test files (`*_test.go`) in the same directory.
- Do not nest multiple levels of directories for a single package.

### Import Formatting
- Group imports logically:
  1. Standard library
  2. Third‑party dependencies
  3. Project‑internal packages
- Use blank imports (`_`) only when necessary for side effects.
- Align import paths on a single line when possible; otherwise, break after commas.

### Formatting
- Run `go fmt ./...` before committing.
- Run `goimports -w ./...` to manage imports automatically.
- Use `golangci-lint run` to enforce formatting and style rules.

### Naming Conventions
- Exported identifiers (functions, structs, variables) use UpperCamelCase.
- Unexported identifiers use lowerCamelCase.
- Package‑level constants use SCREAMING_SNAKE_CASE.
- Function names should be verbs (`HandleMessage`, `SendResponse`).
- Error variable names end with `Err` (`ErrInvalidToken`).

### Type Usage
- Prefer named error variables for common error conditions.
- Avoid exposing internal structs; use getter methods or separate DTOs for external consumption.
- When a function returns multiple values, the error should be the last return value.
- Use context.Context for cancellation and request‑scoped values.

### Error Handling
- Always check `err != nil` after each operation that can fail.
- Wrap errors with `%w` when adding context: `fmt.Errorf("failed to parse token: %w", err)`.
- Do not panic for recoverable errors; return them to the caller.
- Use `defer` to close resources (files, connections) and ensure cleanup.

### Logging
- Use `log.Printf` for structured logging; embed relevant fields (e.g., `UserID`, `ChatID`).
- Do not log secrets (tokens, passwords) or personally identifiable information.
- Prefix logs with a short identifier if the log is part of a larger system.

### Constants and Enumerations
- Define related constant groups in a dedicated `constants` package.
- Use `iota` for numeric enumerations but keep them grouped with explicit values for readability.
- Prefer string representations for user‑facing identifiers; map them via a `map[string]Type`.

### Struct Tags
- Use `json:"field_name"` tags for JSON serialization.
- Use `yaml:"field_name"` tags when using `gopkg.in/yaml.v3` for configuration files.
- Omit empty values; they should not appear in serialized output.

### Testing Practices
- Write table‑driven tests for functions with multiple input combinations.
- Name test functions `Test<Package><Function>` (e.g., `TestHandlerHandle`).
- Isolate unit tests; mock external dependencies using interfaces.
- Run `go test -race ./...` to ensure no data races are present.
- Verify coverage with `go tool cover -func=./...` after test execution.

### Build & Deployment
- Ensure the binary is placed in `bin/` and added to `.gitignore`.
- Use `go.mod` to manage dependencies; run `go mod tidy` before committing.
- Tag releases with semantic versioning (`v1.2.3`) and update `CHANGELOG.md`.
- For Docker, keep the `Dockerfile` minimal: copy only necessary files, use multi‑stage builds if applicable.

### Git Hooks
- Pre‑commit hook runs `golangci-lint run` and `go test ./...` automatically.
- Do not bypass the hook; if a hook fails, fix the issues before committing.

### Additional Tooling
- `staticcheck` and `golangci-lint` are recommended for static analysis.
- `benchstat` can be used to compare benchmark results before and after changes.
- `go vet` should be part of CI pipeline to catch common mistakes.

## CI/CD Integration
- Linting and tests run on every push to `main` or `feature/*` branches.
- Build artifacts are published to the Docker registry on tag creation.
- Codecov is used to enforce a minimum coverage threshold of 80%.

---

*End of document (approx. 150 lines)*