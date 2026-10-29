# kwtSMS Go Client

[![Go Reference](https://pkg.go.dev/badge/github.com/kwtsms/kwtsms-go.svg)](https://pkg.go.dev/github.com/kwtsms/kwtsms-go)
[![CI](https://github.com/kwtsms/kwtsms-go/actions/workflows/ci.yml/badge.svg)](https://github.com/kwtsms/kwtsms-go/actions/workflows/ci.yml)
[![CodeQL](https://github.com/kwtsms/kwtsms-go/actions/workflows/codeql.yml/badge.svg)](https://github.com/kwtsms/kwtsms-go/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kwtsms/kwtsms-go)](https://goreportcard.com/report/github.com/kwtsms/kwtsms-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kwtsms/kwtsms-go)](https://github.com/kwtsms/kwtsms-go)
[![Release](https://img.shields.io/github/v/release/kwtsms/kwtsms-go)](https://github.com/kwtsms/kwtsms-go/releases)

Go client library for the [kwtSMS SMS API](https://www.kwtsms.com). Zero external dependencies. Go 1.18+.

## About kwtSMS

kwtSMS is a Kuwaiti SMS gateway trusted by top businesses to deliver messages anywhere in the world, with private Sender ID, free API testing, non-expiring credits, and competitive flat-rate pricing. Secure, simple to integrate, built to last. Open a free account in under 1 minute, no paperwork or payment required. [Click here to get started](https://www.kwtsms.com/signup/)

## Prerequisites

You need **Go** (1.18 or newer) installed.

### Step 1: Check if Go is installed

```bash
go version
```

If you see a version number, Go is installed. If not:

- **All platforms:** Download from https://go.dev/dl/
- **macOS:** `brew install go`
- **Ubuntu/Debian:** `sudo apt update && sudo apt install golang-go`

### Step 2: Create a project (if you don't have one)

```bash
mkdir my-project && cd my-project
go mod init my-project
```

### Step 3: Install kwtsms

```bash
go get github.com/kwtsms/kwtsms-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    kwtsms "github.com/kwtsms/kwtsms-go"
)

func main() {
    // Load credentials from environment variables or .env file
    sms, err := kwtsms.FromEnv("")
    if err != nil {
        log.Fatal(err)
    }

    // Verify credentials and check balance
    ok, balance, err := sms.Verify()
    if !ok {
        log.Fatalf("verification failed: %v", err)
    }
    fmt.Printf("Balance: %.2f credits\n", balance)

    // Send an SMS
    result, err := sms.Send("96598765432", "Your OTP for MYAPP is: 123456", "")
    if err != nil {
        log.Fatal(err)
    }
    if result.Result == "OK" {
        fmt.Printf("Sent! msg-id: %s, balance: %.2f\n", result.MsgID, result.BalanceAfter)
    } else {
        fmt.Printf("Error: %s\nAction: %s\n", result.Description, result.Action)
    }
}
```

## Setup / Configuration

### CLI Tool

For a standalone command-line tool (all platforms), see [kwtsms-cli](https://github.com/kwtsms/kwtsms-cli).

### Environment variables

Set these environment variables or create a `.env` file in your project root:

```ini
KWTSMS_USERNAME=go_api_user
KWTSMS_PASSWORD=go_api_pass
KWTSMS_SENDER_ID=YOUR-SENDERID
KWTSMS_TEST_MODE=1
KWTSMS_LOG_FILE=kwtsms.log
```

### Constructor

```go
// From environment variables / .env file (recommended)
sms, err := kwtsms.FromEnv("")

// From .env at a custom path
sms, err := kwtsms.FromEnv("/path/to/.env")

// Direct constructor with options
sms, err := kwtsms.New("username", "password",
    kwtsms.WithSenderID("MY-APP"),
    kwtsms.WithTestMode(true),
    kwtsms.WithLogFile("sms.log"),
)
```

Environment variables take priority over `.env` file values.

## Credential Management

**Never hardcode credentials in source code.** Credentials must be changeable without recompiling.

### Environment variables / .env file (recommended for servers)

```go
sms, err := kwtsms.FromEnv("")  // reads KWTSMS_USERNAME, KWTSMS_PASSWORD
```

The `.env` file must be in `.gitignore`. Never commit credentials.

### Constructor injection (for custom config systems)

```go
sms, err := kwtsms.New(
    config.Get("sms_username"),
    config.Get("sms_password"),
)
```

Works with any config source: Vault, AWS Secrets Manager, database, DI containers, etc.

### Admin settings UI (recommended for web apps)

Provide a settings page where an admin can enter API credentials and toggle test mode. Include a "Test Connection" button that calls `Verify()`.

### Remote config / secrets manager (recommended for production)

Load credentials from AWS Secrets Manager, Google Secret Manager, HashiCorp Vault, or your own config API. Credentials rotate without redeployment.

## API Reference

### Verify Credentials

```go
ok, balance, err := sms.Verify()
// ok=true:  credentials valid, balance is the available credit count
// ok=false: err describes the problem with an action to take
```

### Check Balance

```go
balance, err := sms.Balance()
// Returns live balance, or cached value if the API call fails
```

### Send SMS

```go
// Single number
result, err := sms.Send("96598765432", "Hello from Go!", "")

// Multiple numbers (comma-separated)
result, err := sms.Send("96598765432,96512345678", "Bulk message", "")

// Multiple numbers (slice)
result, err := sms.SendMulti(
    []string{"96598765432", "+96512345678", "0096587654321"},
    "Hello everyone!",
    "",
)

// Override sender ID for one call
result, err := sms.Send("96598765432", "Hello", "OTHER-SENDER")
```

**Always save `msg-id` immediately after a successful send.** You need it for status checks and delivery reports. If you do not store it at send time, you cannot retrieve it later.

**Never call `Balance()` after `Send()`.** The send response already includes your updated balance in `result.BalanceAfter`. Save it to your database. The client also caches it internally.

### Send with Retry (ERR028)

```go
// Auto-retries on ERR028 (15-second rate limit), waits 16s between retries
result, err := sms.SendWithRetry("96598765432", "Hello", "", 3)
```

### Bulk Send (>200 numbers)

Sending to more than 200 numbers is handled automatically. The client splits numbers into batches of 200, adds a 0.5s delay between batches, and retries on ERR013 (queue full) with 30s/60s/120s backoff.

```go
numbers := make([]string, 500)
// ... populate numbers ...
result, err := sms.SendMulti(numbers, "Campaign message", "MY-SENDER")
```

### Validate Phone Numbers

```go
result := sms.Validate([]string{"96598765432", "+96512345678", "bad@email.com", "123"})
fmt.Println("Valid:", result.OK)       // valid and routable
fmt.Println("Errors:", result.ER)      // format errors
fmt.Println("No route:", result.NR)    // country not activated
fmt.Println("Rejected:", result.Rejected) // locally rejected (email, too short, etc.)
```

Numbers that fail local validation (empty, email, too short, too long, no digits) are rejected before the API call is made.

### Sender IDs

```go
result := sms.SenderIDs()
if result["result"] == "OK" {
    sids := result["senderids"].([]string)
    fmt.Println("Sender IDs:", sids)
}
```

### Coverage

```go
result := sms.Coverage()
if result["result"] == "OK" {
    fmt.Println("Active coverage:", result)
}
```

### Message Status

```go
result := sms.Status("msg-id-from-send-response")
fmt.Println(result)
```

### Delivery Report (international only)

```go
result := sms.DLR("msg-id-from-send-response")
fmt.Println(result)
```

Kuwait numbers do not support DLR. Only international (non-Kuwait) numbers have delivery reports. Wait at least 5 minutes after sending before checking.

### Cached Balance

```go
// Available after Verify() or successful Send()
if bal := sms.CachedBalance(); bal != nil {
    fmt.Printf("Cached balance: %.2f\n", *bal)
}

// Total purchased credits (available after Verify())
if p := sms.CachedPurchased(); p != nil {
    fmt.Printf("Purchased: %.2f\n", *p)
}
```

## Utility Functions

These are exported for direct use:

```go
// Normalize a phone number: Arabic digits to Latin, strip non-digits, strip leading zeros,
// strip domestic trunk prefix (e.g. 9660559... becomes 966559...)
normalized := kwtsms.NormalizePhone("+965 9876 5432") // "96598765432"
normalized = kwtsms.NormalizePhone("9660559876543")   // "966559876543" (Saudi trunk 0 stripped)

// Validate phone input before sending (includes country-specific rules)
v := kwtsms.ValidatePhoneInput("user@gmail.com")
// v.Valid=false, v.Error="'user@gmail.com' is an email address, not a phone number"

v = kwtsms.ValidatePhoneInput("+96598765432")
// v.Valid=true, v.Normalized="96598765432"

v = kwtsms.ValidatePhoneInput("+96512345678")
// v.Valid=false, v.Error="Invalid Kuwait mobile number: after +965 must start with 4, 5, 6, 9"

// Find the country code from a normalized number
cc := kwtsms.FindCountryCode("96598765432") // "965" (Kuwait)
cc = kwtsms.FindCountryCode("12125551234")  // "1" (USA/Canada)

// Validate against country-specific format rules
err := kwtsms.ValidatePhoneFormat("96598765432") // "" (valid)
err = kwtsms.ValidatePhoneFormat("96512345678")  // "Invalid Kuwait mobile number: ..."

// Look up phone rules for a country (90+ countries covered)
rule := kwtsms.PhoneRules["965"] // {LocalLengths: [8], MobileStartDigits: ["4","5","6","9"]}
name := kwtsms.CountryNames["965"] // "Kuwait"

// Load .env file manually (returns key-value map, never panics)
env := kwtsms.LoadEnvFile("/path/to/.env")

// Clean a message: strip emojis, HTML, control chars, convert Arabic digits
cleaned := kwtsms.CleanMessage("Hello  <b>World</b> 123")
// "Hello  World 123"

// API error map (for building custom error UIs)
action := kwtsms.APIErrors["ERR003"]
// "Wrong API username or password. Check KWTSMS_USERNAME and KWTSMS_PASSWORD..."

// Enrich an error response with action guidance
enriched := kwtsms.EnrichError(apiResponse)
```

## Phone Validation Rules

`ValidatePhoneInput()` and `Send()` automatically validate numbers against country-specific rules for 90+ countries. Validation checks:

1. **Local number length**: each country has specific valid lengths (e.g., Kuwait: 8 digits after +965)
2. **Mobile prefix**: each country has valid starting digits for mobile numbers (e.g., Kuwait: 4, 5, 6, 9)
3. **Domestic trunk prefix**: automatically stripped during normalization (e.g., 9660559... becomes 966559...)

Numbers from countries not in the rules table pass through with generic E.164 validation (7-15 digits).

GCC coverage:

| Country | Code | Local Digits | Mobile Prefixes |
|---------|------|-------------|-----------------|
| Kuwait | 965 | 8 | 4x, 5x, 6x, 9x |
| Saudi Arabia | 966 | 9 | 5x |
| UAE | 971 | 9 | 5x |
| Bahrain | 973 | 8 | 3x, 6x |
| Qatar | 974 | 8 | 3x, 5x, 6x, 7x |
| Oman | 968 | 8 | 7x, 9x |

The full rules table (`kwtsms.PhoneRules`) covers GCC, Levant, North Africa, Europe, Americas, Asia, Africa, and Oceania. See `phone.go` for the complete list.

## Input Sanitization

The `CleanMessage()` function runs automatically before every send. It prevents the most common cause of "message sent but not received" support tickets:

| Content | Problem | Fix |
|---------|---------|-----|
| Emojis | Message stuck in queue indefinitely, credits wasted, no error returned | Stripped before send |
| Hidden control characters (BOM, zero-width spaces, soft hyphens) | Spam filter rejection or queue stuck, common in text from Word/PDF/rich editors | Stripped before send |
| Arabic/Hindi numerals in body | OTP codes and amounts may render inconsistently | Converted to Latin digits |
| HTML tags | ERR027, message rejected | Stripped before send |
| C0/C1 control characters | Unprintable binary from copy-pasting terminals or binary content | Stripped (except newlines and tabs) |
| Directional marks (LTR, RTL, LRE, etc.) | Introduced by rich-text editors and RTL-aware apps | Stripped before send |

Arabic letters and Arabic text are fully preserved. Only digits are converted, invisible characters are removed, and emojis are stripped.

`Send()` calls `CleanMessage()` automatically, but you can also call it directly to preview what the API will receive:

```go
cleaned := kwtsms.CleanMessage(userInput)
if strings.TrimSpace(cleaned) == "" {
    // Message was only emojis or control characters
    return fmt.Errorf("message is empty after cleaning")
}
```

## Exports

### Types

```go
kwtsms.KwtSMS           // Client struct
kwtsms.SendResult       // Send response
kwtsms.BulkSendResult   // Bulk send response (>200 numbers)
kwtsms.ValidateResult   // Validate response
kwtsms.InvalidEntry     // Rejected phone number with error message
kwtsms.BatchError       // Error from a single batch in bulk send
kwtsms.PhoneValidation  // Result of ValidatePhoneInput
kwtsms.PhoneRule        // Country-specific phone validation rule
kwtsms.Option           // Functional option for New()
```

### Functions

```go
kwtsms.NormalizePhone()      // Normalize phone: Arabic digits, strip non-digits, trunk prefix
kwtsms.ValidatePhoneInput()  // Validate phone input with country-specific rules
kwtsms.FindCountryCode()     // Extract country code from normalized number
kwtsms.ValidatePhoneFormat() // Validate against country-specific length and prefix rules
kwtsms.CleanMessage()        // Strip emojis, HTML, control chars, convert Arabic digits
kwtsms.EnrichError()         // Add action guidance to API error response
kwtsms.LoadEnvFile()         // Parse .env file into key-value map
```

### Variables

```go
kwtsms.APIErrors     // map[string]string: error code to action message (33 codes)
kwtsms.PhoneRules    // map[string]PhoneRule: country code to validation rules (90+ countries)
kwtsms.CountryNames  // map[string]string: country code to human-readable name
```

## Error Handling

Every API error includes a developer-friendly `action` field explaining what to do. All 33 kwtSMS error codes are mapped.

```go
result, _ := sms.Send("96598765432", "Hello", "")
if result.Result == "ERROR" {
    fmt.Println("Code:", result.Code)             // "ERR003"
    fmt.Println("Description:", result.Description) // "Authentication error..."
    fmt.Println("Action:", result.Action)           // "Wrong API username or password. Check..."
}
```

### User-Facing Error Messages

Raw API errors are for developers, not end users. Map them for your UI:

| Situation | API error | Show to user |
|-----------|-----------|--------------|
| Invalid phone | ERR006, ERR025 | "Please enter a valid phone number in international format (e.g., +965 9876 5432)." |
| Wrong credentials | ERR003 | "SMS service is temporarily unavailable. Please try again later." |
| No balance | ERR010, ERR011 | "SMS service is temporarily unavailable. Please try again later." |
| Country not supported | ERR026 | "SMS delivery to this country is not available. Please contact support." |
| Rate limited | ERR028 | "Please wait a moment before requesting another code." |
| Message rejected | ERR031, ERR032 | "Your message could not be sent. Please try again with different content." |
| Network error | timeout | "Could not connect to SMS service. Check your internet connection." |
| Queue full | ERR013 | "SMS service is busy. Please try again in a few minutes." |

**Key principle:** user-recoverable errors (bad phone, rate limited) get a helpful message. System-level errors (auth, balance, network) get a generic message + log the real error + alert the admin.

## Phone Number Formats

All formats are accepted and normalized automatically:

| Input | Normalized | Valid? |
|-------|-----------|--------|
| `96598765432` | `96598765432` | Yes |
| `+96598765432` | `96598765432` | Yes |
| `0096598765432` | `96598765432` | Yes |
| `965 9876 5432` | `96598765432` | Yes |
| `965-9876-5432` | `96598765432` | Yes |
| `(965) 98765432` | `96598765432` | Yes |
| `965.9876.5432` | `96598765432` | Yes |
| `٩٦٥٩٨٧٦٥٤٣٢` | `96598765432` | Yes |
| `۹۶۵۹۸۷۶۵۴۳۲` | `96598765432` | Yes |
| `+٩٦٥٩٨٧٦٥٤٣٢` | `96598765432` | Yes |
| `٠٠٩٦٥٩٨٧٦٥٤٣٢` | `96598765432` | Yes |
| `٩٦٥ ٩٨٧٦ ٥٤٣٢` | `96598765432` | Yes |
| `٩٦٥-٩٨٧٦-٥٤٣٢` | `96598765432` | Yes |
| `965٩٨٧٦٥٤٣٢` | `96598765432` | Yes |
| `123456` (too short) | rejected | No |
| `user@gmail.com` | rejected | No |

Numbers must be in international format with country code. Arabic-Indic (U+0660-U+0669) and Persian (U+06F0-U+06F9) digits are converted to Latin automatically.

## Test Mode

**Test mode** (`KWTSMS_TEST_MODE=1`) sends your message to the kwtSMS queue but does NOT deliver it to the handset. No SMS credits are consumed. Use during development and testing.

**Live mode** (`KWTSMS_TEST_MODE=0`) delivers the message for real and deducts credits.

```go
// Enable test mode via constructor
sms, _ := kwtsms.New("user", "pass", kwtsms.WithTestMode(true))

// Or via environment variable / .env file
// KWTSMS_TEST_MODE=1
```

Always develop in test mode and switch to live only when ready for production. Test messages appear in the **Sending Queue** at kwtsms.com. Delete them from the queue to recover any tentatively held credits.

## Sender ID

A **Sender ID** is the name that appears as the sender on the recipient's phone (e.g., "MY-APP" instead of a random number).

| | Promotional | Transactional |
|--|-------------|---------------|
| **Use for** | Bulk SMS, marketing, offers | OTP, alerts, notifications |
| **DND numbers** | Blocked/filtered, credits lost | Bypasses DND |
| **Speed** | May have delays | Priority delivery |
| **Cost** | 10 KD one-time | 15 KD one-time |

`KWT-SMS` is the shared test sender. It causes delays and is blocked on Virgin Kuwait. **Never use in production.** Register a private sender ID at kwtsms.com.

For OTP/authentication, you **must** use a Transactional sender ID. Promotional sender IDs are filtered by DND (Do Not Disturb) on Zain and Ooredoo, meaning OTP messages silently fail and credits are still deducted.

Sender ID is **case sensitive**: `Kuwait` is not the same as `KUWAIT`.

Registration takes ~5 working days for Kuwait and 1-2 months for international.

## Best Practices

### 1. Validate before calling the API

The #1 cause of wasted API calls: sending invalid input and letting the API reject it. Validate locally first:

```go
// BAD: wastes an API call on every invalid input
result, _ := sms.Send(userInput, message, "")

// GOOD: validate locally, only hit API with clean input
v := kwtsms.ValidatePhoneInput(userInput)
if !v.Valid {
    return fmt.Errorf("invalid phone: %s", v.Error)
}
cleaned := kwtsms.CleanMessage(message)
if strings.TrimSpace(cleaned) == "" {
    return fmt.Errorf("message is empty after cleaning")
}
result, _ := sms.Send(v.Normalized, message, "")
```

The `Send()` method does validate and clean internally, but checking first lets you return errors to the user immediately without a network round-trip.

### 2. Cache coverage at startup

Call `Coverage()` once at application startup and cache the active country prefixes. Before every send, check the number's country prefix against the cached list. If the country is not active, return an error immediately without hitting the API:

```go
coverage := sms.Coverage()
// Cache the active country prefixes, check before every send
// "SMS delivery to [country] is not available on this account."
```

### 3. Save balance-after and msg-id

```go
if result.Result == "OK" {
    db.SaveBalance(result.BalanceAfter)   // track balance without extra API calls
    db.SaveMsgID(result.MsgID)            // needed for Status() and DLR() later
}
```

Set up low-balance alerts (e.g., when balance drops below 50 credits). Before bulk sends, estimate credit cost (number of recipients x pages per message) and warn if balance is insufficient.

### 4. Sender ID

`KWT-SMS` is the shared test sender. It causes delays and is blocked on Virgin Kuwait. **Never use in production.** Register a private sender ID at kwtsms.com.

For OTP/authentication, you **must** use a Transactional sender ID. Promotional sender IDs are filtered by DND (Do Not Disturb) on Zain and Ooredoo, meaning OTP messages silently fail and credits are still deducted.

### 5. OTP implementation

- Always include the app/company name: `"Your OTP for APPNAME is: 123456"`
- Resend timer: minimum 3-4 minutes (KNET standard is 4 minutes)
- OTP expiry: 3-5 minutes
- Generate a new code on resend, invalidate all previous codes
- Send to one number per request (avoid ERR028 batch rejection)
- Use a Transactional sender ID (not Promotional)

### 6. Rate limiting

Wait at least **15 seconds** before sending to the same number again (ERR028). The entire request is rejected if any number in a batch triggers this, even if other numbers are fine.

### 7. Monitoring and alerting

Set up alerts for:
- Failed sends: sudden increase in error responses
- Balance depletion: rapid decrease or approaching zero
- Error rate spikes: especially ERR003 (credentials), ERR010/ERR011 (balance), ERR028 (rate limit)
- Queue buildup: messages stuck in kwtSMS queue (check via dashboard)

### 8. Keep libraries updated

Monitor for security patches and updates to the kwtSMS client library. Subscribe to kwtSMS announcements for API changes.

### 9. Compliance

Stay informed about local telecom regulations regarding sender IDs, message content, and user consent. Promotional SMS may require opt-in consent from recipients. Different countries have different rules: check before enabling international coverage.

## Timestamps

`unix-timestamp` values in API responses are in **GMT+3 (Asia/Kuwait)** server time, not UTC. Convert when storing or displaying. Log timestamps written by the client are always UTC ISO-8601.

## Security Checklist

Before going live:

```
[ ] Bot protection enabled (CAPTCHA for web apps)
[ ] Rate limit per phone number (max 3-5/hour)
[ ] Rate limit per IP address (max 10-20/hour)
[ ] Rate limit per user/session if authenticated
[ ] Monitoring/alerting on abuse patterns
[ ] Admin notification on low balance
[ ] Test mode OFF (KWTSMS_TEST_MODE=0)
[ ] Private Sender ID registered (not KWT-SMS)
[ ] Transactional Sender ID for OTP (not promotional)
```

Without rate limiting, a bot can drain your entire SMS balance in minutes.

## What's Handled Automatically

- **Phone normalization**: `+`, `00`, spaces, dashes, dots, parentheses stripped. Arabic-Indic digits converted. Leading zeros removed.
- **Duplicate phone removal**: If the same number appears multiple times (in different formats), it is sent only once.
- **Message cleaning**: Emojis removed (surrogate-pair safe). Hidden control characters (BOM, zero-width spaces, directional marks) removed. HTML tags stripped. Arabic-Indic digits in message body converted to Latin.
- **Batch splitting**: More than 200 numbers are automatically split into batches of 200 with 0.5s delay between batches.
- **ERR013 retry**: Queue-full errors are automatically retried up to 3 times with exponential backoff (30s / 60s / 120s).
- **Error enrichment**: Every API error response includes an `action` field with a developer-friendly fix hint.
- **Credential masking**: Passwords are always masked as `***` in log files. Never exposed.
- **Never throws**: All public methods return structured error objects. They never panic on API errors.

## Examples

See the [examples/](examples/) directory for runnable code:

| Example | Description |
|---------|-------------|
| [00-raw-api](examples/00-raw-api/) | Call every kwtSMS endpoint using only the Go standard library (no dependencies) |
| [01-basic-usage](examples/01-basic-usage/) | Load credentials, verify, send SMS, print result |
| [02-otp-flow](examples/02-otp-flow/) | Generate OTP, validate phone, send, save msg-id |
| [03-bulk-sms](examples/03-bulk-sms/) | Send to multiple numbers with mixed formats |
| [04-http-handler](examples/04-http-handler/) | HTTP endpoint for sending SMS with validation |
| [05-error-handling](examples/05-error-handling/) | Handle all error types with user-facing messages |
| [06-otp-production](examples/06-otp-production/) | Production OTP server: rate limiting, expiry, resend cooldown, user-facing errors |

## Testing

```bash
# Unit + mocked API tests (no credentials needed)
go test -v ./...

# With race detector
go test -race ./...

# Integration tests (hits live API with test_mode=true, no credits consumed)
GO_USERNAME=go_user GO_PASSWORD=go_pass go test -v -tags integration ./...
```

## JSONL Logging

Every API call is logged to `kwtsms.log` (configurable) as one JSON line. Passwords are always masked as `***`. Timestamps are UTC ISO-8601.

```json
{"ts":"2026-03-06T12:00:00Z","endpoint":"send","request":{"username":"go_user","password":"***","sender":"MY-APP","mobile":"96598765432","message":"Hello","test":"1"},"response":{"result":"OK","msg-id":"abc123"},"ok":true}
```

Logging never crashes the main flow. Disk errors are silently ignored.

## FAQ

**1. My message was sent successfully (result: OK) but the recipient didn't receive it. What happened?**

Check the **Sending Queue** at [kwtsms.com](https://www.kwtsms.com/login/). If your message is stuck there, it was accepted by the API but not dispatched. Common causes are emoji in the message, hidden characters from copy-pasting, or spam filter triggers. Delete it from the queue to recover your credits. Also verify that `test` mode is off (`KWTSMS_TEST_MODE=0`). Test messages are queued but never delivered.

**2. What is the difference between Test mode and Live mode?**

**Test mode** (`KWTSMS_TEST_MODE=1`) sends your message to the kwtSMS queue but does NOT deliver it to the handset. No SMS credits are consumed. Use during development. **Live mode** (`KWTSMS_TEST_MODE=0`) delivers the message for real and deducts credits. Always develop in test mode and switch to live only when ready for production.

**3. What is a Sender ID and why should I not use "KWT-SMS" in production?**

A **Sender ID** is the name that appears as the sender on the recipient's phone (e.g., "MY-APP" instead of a random number). `KWT-SMS` is a shared test sender. It causes delivery delays, is blocked on Virgin Kuwait, and should never be used in production. Register your own private Sender ID through your kwtSMS account. For OTP/authentication messages, you need a **Transactional** Sender ID to bypass DND (Do Not Disturb) filtering.

**4. I'm getting ERR003 "Authentication error". What's wrong?**

You are using the wrong credentials. The API requires your **API username and API password**, NOT your account mobile number. Log in to [kwtsms.com](https://www.kwtsms.com/login/), go to Account > API settings, and check your API credentials. Also make sure you are using POST (not GET) and `Content-Type: application/json`.

**5. Can I send to international numbers (outside Kuwait)?**

International sending is **disabled by default** on kwtSMS accounts. [Log in to your kwtSMS account](https://www.kwtsms.com/login/) and add coverage for the country prefixes you need. Use `Coverage()` to check which countries are currently active on your account. Be aware that activating international coverage increases exposure to automated abuse. Implement rate limiting and CAPTCHA before enabling.

## Help & Support

- **[kwtSMS FAQ](https://www.kwtsms.com/faq/)**: Answers to common questions about credits, sender IDs, OTP, and delivery
- **[kwtSMS Support](https://www.kwtsms.com/support.html)**: Open a support ticket or browse help articles
- **[Contact kwtSMS](https://www.kwtsms.com/#contact)**: Reach the kwtSMS team directly for Sender ID registration and account issues
- **[API Documentation (PDF)](https://www.kwtsms.com/doc/KwtSMS.com_API_Documentation_v41.pdf)**: kwtSMS REST API v4.1 full reference
- **[Best Practices](https://www.kwtsms.com/articles/sms-api-implementation-best-practices.html)**: SMS API implementation best practices
- **[Integration Test Checklist](https://www.kwtsms.com/articles/sms-api-integration-test-checklist.html)**: Pre-launch testing checklist
- **[Sender ID Help](https://www.kwtsms.com/sender-id-help.html)**: Sender ID registration and guidelines
- **[kwtSMS Dashboard](https://www.kwtsms.com/login/)**: Recharge credits, buy Sender IDs, view message logs, manage coverage
- **[Other Integrations](https://www.kwtsms.com/integrations.html)**: Plugins and integrations for other platforms and languages
- **[Library Issues](https://github.com/kwtsms/kwtsms-go/issues)**: Report bugs or request features for this Go client

## License

MIT. See [LICENSE](LICENSE).

