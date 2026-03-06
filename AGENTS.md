# AGENTS.md

This file contains guidelines and commands for agentic coding agents working in this repository.

---

## Max Bot (Go)

### Build/Test/Lint Commands

This is a Go project with the following commands:

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

### Code Style Guidelines

#### File Organization
- Keep each package in its own directory (`handlers`, `bot`, `utils`).
- Place implementation files (`*.go`) alongside test files (`*_test.go`) in the same directory.
- Do not nest multiple levels of directories for a single package.

#### Import Formatting
- Group imports logically:
  1. Standard library
  2. Third‑party dependencies
  3. Project‑internal packages
- Use blank imports (`_`) only when necessary for side effects.
- Align import paths on a single line when possible; otherwise, break after commas.

#### Formatting
- Run `go fmt ./...` before committing.
- Run `goimports -w ./...` to manage imports automatically.
- Use `golangci-lint run` to enforce formatting and style rules.

#### Naming Conventions
- Exported identifiers (functions, structs, variables) use UpperCamelCase.
- Unexported identifiers use lowerCamelCase.
- Package‑level constants use SCREAMING_SNAKE_CASE.
- Function names should be verbs (`HandleMessage`, `SendResponse`).
- Error variable names end with `Err` (`ErrInvalidToken`).

#### Type Usage
- Prefer named error variables for common error conditions.
- Avoid exposing internal structs; use getter methods or separate DTOs for external consumption.
- When a function returns multiple values, the error should be the last return value.
- Use context.Context for cancellation and request‑scoped values.

#### Error Handling
- Always check `err != nil` after each operation that can fail.
- Wrap errors with `%w` when adding context: `fmt.Errorf("failed to parse token: %w", err)`.
- Do not panic for recoverable errors; return them to the caller.
- Use `defer` to close resources (files, connections) and ensure cleanup.

#### Logging
- Use `log.Printf` for structured logging; embed relevant fields (e.g., `UserID`, `ChatID`).
- Do not log secrets (tokens, passwords) or personally identifiable information.
- Prefix logs with a short identifier if the log is part of a larger system.

#### Constants and Enumerations
- Define related constant groups in a dedicated `constants` package.
- Use `iota` for numeric enumerations but keep them grouped with explicit values for readability.
- Prefer string representations for user‑facing identifiers; map them via a `map[string]Type`.

#### Struct Tags
- Use `json:"field_name"` tags for JSON serialization.
- Use `yaml:"field_name"` tags when using `gopkg.in/yaml.v3` for configuration files.
- Omit empty values; they should not appear in serialized output.

#### Testing Practices
- Write table‑driven tests for functions with multiple input combinations.
- Name test functions `Test<Package><Function>` (e.g., `TestHandlerHandle`).
- Isolate unit tests; mock external dependencies using interfaces.
- Run `go test -race ./...` to ensure no data races are present.
- Verify coverage with `go tool cover -func=./...` after test execution.

#### Build & Deployment
- Ensure the binary is placed in `bin/` and added to `.gitignore`.
- Use `go.mod` to manage dependencies; run `go mod tidy` before committing.
- Tag releases with semantic versioning (`v1.2.3`) and update `CHANGELOG.md`.
- For Docker, keep the `Dockerfile` minimal: copy only necessary files, use multi‑stage builds if applicable.

#### Git Hooks
- Pre‑commit hook runs `golangci-lint run` and `go test ./...` automatically.
- Do not bypass the hook; if a hook fails, fix the issues before committing.

#### Additional Tooling
- `staticcheck` and `golangci-lint` are recommended for static analysis.
- `benchstat` can be used to compare benchmark results before and after changes.
- `go vet` should be part of CI pipeline to catch common mistakes.

#### CI/CD Integration
- Linting and tests run on every push to `main` or `feature/*` branches.
- Build artifacts are published to the Docker registry on tag creation.
- Codecov is used to enforce a minimum coverage threshold of 80%.

---

## Joomla Article Manager (Python)

### Environment Setup
```bash
# Install dependencies
pip install -r requirements.txt

# Or install as a package
pip install -e .
```

### Running Scripts
```bash
# Run the main Joomla extractor
python3 joomla_extractor.py

# Run improved extractor with TinyMCE support
python3 improved_extractor.py

# Run article updater
python3 update_article_remote.py

# Get article content
python3 get_article_content.py 1025 schedule.html
```

### Testing
This project does not have formal test suites. Manual testing is performed by:
1. Running the extractor scripts
2. Verifying JSON output files
3. Checking database operations (if configured)

### Linting and Formatting
```bash
# Check code style (if flake8 is installed)
flake8 . --max-line-length=120

# Format code (if black is installed)
black . --line-length=120

# Sort imports (if isort is installed)
isort .
```

### Code Style Guidelines

#### Python Version and Dependencies
- Target Python 3.6+
- Use `requests>=2.25.1`, `beautifulsoup4>=4.9.3`, `mysql-connector-python>=8.0.25`
- All scripts should import from `config.py` for configuration

#### Import Organization
```python
# Standard library imports first
import sys
import json
import time
from urllib.parse import urljoin, urlparse

# Third-party imports next
import requests
from bs4 import BeautifulSoup

# Local imports last
import config
```

#### Class Structure
- All main functionality should be encapsulated in classes
- Class names should be descriptive: `JoomlaContentExtractor`, `JoomlaArticleUpdater`
- Initialize with session and configuration in `__init__`
- Use instance variables for session, URLs, credentials, and state

#### Session Management
```python
class ExampleClass:
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({'User-Agent': config.USER_AGENT})
        self.site_url = config.SITE_URL
        self.admin_url = config.ADMIN_URL
        self.username = config.USERNAME
        self.password = config.PASSWORD
        self.csrf_token = None
        self.authenticated = False
```

#### Error Handling
- Use try/except blocks for network operations
- Handle `requests.exceptions.RequestException` for HTTP errors
- Use `response.raise_for_status()` to catch HTTP errors
- Print meaningful error messages in Russian (as per project convention)

#### CSRF Token Handling
- Always retrieve CSRF token from admin page before POST requests
- Check both JSON script options and hidden form fields
- Store token in instance variable for reuse

#### Configuration
- Never hardcode credentials or URLs
- Use `config.py` for all configuration
- Respect `.gitignore` to exclude sensitive files
- Use environment variables for production deployments

#### Naming Conventions
- Classes: PascalCase (e.g., `JoomlaContentExtractor`)
- Functions/methods: snake_case (e.g., `get_csrf_token`, `login`)
- Variables: snake_case (e.g., `site_url`, `csrf_token`)
- Constants: UPPER_SNAKE_CASE (e.g., `REQUEST_TIMEOUT`, `VERIFY_SSL`)

#### File Organization
- Main functionality in `joomla_extractor.py` and `improved_extractor.py`
- Article update scripts in `update_article_*.py` files
- Database operations in `update_db.py`
- Utility scripts for specific tasks
- Configuration in `config.py` and `config_example.py`

#### JSON Output
- Save results in `joomla_articles.json`
- Include article metadata: ID, title, introtext, fulltext, category, status, dates
- Use UTF-8 encoding for Russian characters

#### Security Considerations
- Never commit `config.py` with real credentials
- Use HTTPS connections (`VERIFY_SSL = True`)
- Limit database permissions if direct access is used
- Rotate credentials regularly in production

#### Russian Language Convention
- Comments and print statements should be in Russian
- Follow existing Russian terminology in the codebase
- Use proper Russian grammar in user-facing messages

#### Session Configuration
- Respect `SESSION_TIMEOUT` and `REQUEST_TIMEOUT` from config
- Use `USER_AGENT` string from config for all requests
- Handle SSL verification based on `VERIFY_SSL` setting

#### Database Operations
- Use `mysql-connector-python` for direct database access
- Follow Joomla database schema for articles and content
- Handle UTF-8 encoding for Russian text in database operations

#### Documentation
- Include docstrings for all classes and methods
- Use Russian for documentation comments
- Reference Joomla version (3.10.12) in documentation
- Include installation and usage instructions in README.md