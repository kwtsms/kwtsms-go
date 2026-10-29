# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.1] - 2026-10-29

### Changed
- The repository moved to the kwtSMS GitHub organization. The module path is now `github.com/kwtsms/kwtsms-go`; update your imports. Old versions keep working from the old path through GitHub's redirect. No code changes.

## [0.4.0] - 2026-03-13

### Added

- Country-specific phone validation: `PhoneRules` (90+ countries), `FindCountryCode()`, `ValidatePhoneFormat()`, `CountryNames`
- Domestic trunk prefix stripping in `NormalizePhone()` (e.g., `9660559...` becomes `966559...`)
- `ValidatePhoneInput()` now validates against country-specific length and mobile prefix rules
- GitGuardian secrets scanning workflow on push and PR

### Removed

- CLI tool (`cmd/kwtsms/`): moved to dedicated repo [kwtsms-cli](https://github.com/kwtsms/kwtsms-cli)
- CLI cross-compilation from release workflow

### Fixed

- All golangci-lint errcheck, staticcheck, and unused warnings resolved

[0.4.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.4.0

## [0.3.0] - 2026-03-07

### Added

- `WithHTTPClient` option for dependency injection (enables testable HTTP clients)
- Enhanced setup wizard: loads existing `.env` defaults, fetches sender IDs for selection, log file prompt with "off" option, test/live mode menu, newline sanitization
- CLI tests: 37 unit tests covering all commands, setup wizard flows, auto-setup
- Bulk send integration tests: 250-number send with balance verification (library and CLI)
- Raw API example (example 00): all 7 endpoints with zero dependencies
- `\r`/`\n` stripping in `New()` constructor to prevent Windows line ending corruption in credentials

### Changed

- `request()` is now a method on `KwtSMS` (was a standalone function), uses per-instance HTTP client
- `LoadEnvFile` exported for CLI and external consumers
- CLI refactored to `app` struct pattern for testability (injected I/O, exit codes instead of `os.Exit`)
- Setup wizard sanitizes credentials before verification (was after)

## [0.2.0] - 2026-03-06

### Added

- Production OTP example with rate limiting, expiry, resend cooldown, and user-facing errors
- CodeQL badge and Go Version badge in README
- Complete README: About kwtSMS, Prerequisites, Input Sanitization, Phone Number Formats, Test Mode, What's Handled Automatically, FAQ, Help & Support sections
- Arabic-Indic and Persian digit examples in Phone Number Formats table

### Changed

- All placeholder credentials now use `go_` prefix (e.g., `go_api_user`, `go_api_pass`)
- Help & Support links match official kwtSMS format with Library Issues link
- Removed duplicate Publishing and Links sections from README

## [0.1.0] - 2026-03-06

### Added

- `KwtSMS` client with `New()` and `FromEnv()` constructors
- `Verify()` to test credentials and check balance
- `Balance()` to get current balance
- `Send()` and `SendMulti()` for single and multi-number sends
- `SendWithRetry()` for automatic ERR028 retry
- Bulk send support (>200 numbers auto-batched with ERR013 retry)
- `Validate()` to validate phone numbers via the API
- `SenderIDs()` to list registered sender IDs
- `Coverage()` to list active country prefixes
- `Status()` to check message delivery status
- `DLR()` to retrieve delivery reports (international only)
- `NormalizePhone()` utility: Arabic/Persian digits, strip +/00/spaces/dashes
- `ValidatePhoneInput()` utility: catches email, empty, too short, too long, no digits
- `CleanMessage()` utility: strips emojis, HTML, BOM, zero-width chars, C0/C1 controls
- `APIErrors` map with all 33 error codes and developer-friendly action messages
- `EnrichError()` to add action guidance to API error responses
- Phone number deduplication before send
- Thread-safe cached balance with `sync.Mutex`
- JSONL logging with password masking
- `.env` file parser with environment variable priority
- CLI with setup, verify, balance, send, validate, senderid, coverage, status, dlr commands
- `kwtsms setup` interactive wizard: prompts for credentials, verifies them, writes `.env` with 0600 permissions
- Unit tests for phone normalization, message cleaning, error enrichment
- Mocked HTTP API tests for all error codes and endpoints
- Integration tests (build tag `integration`, uses `GO_USERNAME`/`GO_PASSWORD`)
- GitHub Actions CI (Go 1.18-1.22, race detector, integration on tag)
- CodeQL security scanning and Dependabot dependency updates
- Five runnable examples: basic usage, OTP flow, bulk SMS, HTTP handler, error handling

[0.3.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.1.0
