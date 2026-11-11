# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-11-06

### Added

- Client identifies itself with name and version; `WithAppName` option.

### Changed

- `SendWithRetry()` retry options updated.

## [0.4.1] - 2026-10-29

### Changed
- The repository moved to the kwtSMS GitHub organization. The module path is now `github.com/kwtsms/kwtsms-go`; update your imports.

## [0.4.0] - 2026-03-13

### Added

- Country-specific phone validation.
- Trunk prefix handling in phone normalization.
- Secrets scanning workflow.

### Removed

- Built-in CLI, moved to [kwtsms-cli](https://github.com/kwtsms/kwtsms-cli).

### Fixed

- Bug fixes.

[0.4.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.4.0

## [0.3.0] - 2026-03-07

### Added

- `WithHTTPClient` option.
- Enhanced setup wizard.
- Bulk send integration tests.
- Raw API example.

### Changed

- `LoadEnvFile` exported.

## [0.2.0] - 2026-03-06

### Added

- Production OTP example.
- Complete README.

### Changed

- Placeholder credentials now use the `go_` prefix.

## [0.1.0] - 2026-03-06

### Added

- `KwtSMS` client with `New()` and `FromEnv()` constructors
- `Verify()`, `Balance()`, `Send()`, `SendMulti()`, `SendWithRetry()`, `Validate()`, `SenderIDs()`, `Coverage()`, `Status()` and `DLR()`
- Bulk send with automatic batching
- Phone and message utilities
- Error code map with developer-friendly messages
- JSONL logging
- `.env` file parser
- CLI with setup wizard
- Unit, mocked and integration tests
- GitHub Actions CI
- Five runnable examples

[0.3.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/kwtsms/kwtsms-go/releases/tag/v0.1.0
