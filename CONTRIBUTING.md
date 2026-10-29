# Contributing to kwtsms-go

## Prerequisites

- Go 1.18 or later
- A kwtSMS account (for integration tests only)

## Setup

```bash
git clone https://github.com/kwtsms/kwtsms-go.git
cd kwtsms-go
go mod download
```

## Running Tests

```bash
# Unit + mocked API tests (no credentials needed)
go test -v ./...

# With race detector
go test -race ./...

# Integration tests (needs real API credentials, test_mode=true)
GO_USERNAME=go_user GO_PASSWORD=go_pass go test -v -tags integration ./...
```

## Project Structure

```
kwtsms.go           Main client: New(), FromEnv(), Verify(), Send(), etc.
phone.go            NormalizePhone(), ValidatePhoneInput()
message.go          CleanMessage()
errors.go           APIErrors map, EnrichError(), type definitions
request.go          HTTP POST helper
logger.go           JSONL logger
env.go              .env file parser
cmd/kwtsms/main.go  CLI binary
*_test.go           Unit, mocked, and integration tests
```

## Branch Naming

- `feature/short-description` for new features
- `fix/short-description` for bug fixes
- `docs/short-description` for documentation changes

## Pull Request Checklist

- [ ] All existing tests pass: `go test ./...`
- [ ] New code has tests
- [ ] `go vet ./...` reports no issues
- [ ] Race detector passes: `go test -race ./...`
- [ ] No external dependencies added (zero-dependency policy)
- [ ] Exported functions have Go doc comments
- [ ] Error messages follow the existing pattern (structured, with `action` field)

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Exported names: `CamelCase` (e.g., `NormalizePhone`, `ValidatePhoneInput`)
- Unexported names: `camelCase` (e.g., `loadEnvFile`, `writeLog`)
- Return `(result, error)` tuples for fallible operations
- Never panic. Return errors.
- Password must be masked as `***` in all logs

## Zero-Dependency Policy

The library uses only the Go standard library. Do not add external dependencies. Go stdlib provides everything needed: `net/http`, `encoding/json`, `os`, `regexp`, `strings`, `unicode`, `sync`, `time`.

## Releasing

1. Update `Version` constant in `kwtsms.go`
2. Update `CHANGELOG.md`
3. Commit and push to `main`
4. Tag the release: `git tag v0.X.0 && git push origin v0.X.0`
5. Create a GitHub release with notes
6. pkg.go.dev indexes the new version automatically
