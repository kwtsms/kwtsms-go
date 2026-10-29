# Example 00: Raw API Calls (No Library)

This example shows how to call every kwtSMS API endpoint using **only the Go standard library**. No external packages, no kwtsms-go client. Copy any function into your own code and start sending SMS immediately.

## When to use this

- You want to understand exactly what the API expects and returns
- You prefer zero dependencies
- You are building your own wrapper or integrating into an existing HTTP stack
- You want a quick reference for endpoint URLs, payloads, and responses

For production use, the [kwtsms-go client library](https://github.com/kwtsms/kwtsms-go) handles phone normalization, message cleaning, bulk batching, retry logic, error enrichment, and JSONL logging automatically.

## Endpoints covered

| # | Function | Endpoint | Description |
|---|----------|----------|-------------|
| 1 | `checkBalance()` | `POST /API/balance/` | Verify credentials, get available/purchased credits |
| 2 | `listSenderIDs()` | `POST /API/senderid/` | List sender IDs registered on your account |
| 3 | `listCoverage()` | `POST /API/coverage/` | List active country prefixes |
| 4 | `validateNumbers()` | `POST /API/validate/` | Check phone numbers before sending |
| 5 | `sendSMS()` | `POST /API/send/` | Send SMS to one or more numbers |
| 6 | `checkStatus()` | `POST /API/status/` | Check delivery status of a sent message |
| 7 | `checkDLR()` | `POST /API/dlr/` | Get delivery report (international numbers only) |

## Setup

1. Open `main.go` and edit the configuration variables at the top:

```go
var (
    username = "your_api_username"   // your kwtSMS API username
    password = "your_api_password"   // your kwtSMS API password
    senderID = "KWT-SMS"             // sender name shown on recipient's phone
    testMode = "1"                   // "1" = test (no delivery), "0" = live
)
```

Find your API credentials at: [kwtsms.com](https://www.kwtsms.com/login/) > Account > API settings.

**These are NOT your account mobile number.** They are separate API credentials.

2. Run the example:

```bash
cd examples/00-raw-api
go run main.go
```

## How it works

### The helper function

Every kwtSMS endpoint uses `POST` with a JSON body and returns JSON. One helper handles all of them:

```go
func callAPI(endpoint string, payload map[string]any) (map[string]any, error)
```

- Sends `POST` to `https://www.kwtsms.com/API/<endpoint>/`
- Sets `Content-Type: application/json` and `Accept: application/json`
- 15-second timeout
- Returns the parsed JSON response as a map

### Step-by-step walkthrough

#### 1. Balance (verify credentials)

```go
data, err := callAPI("balance", map[string]any{
    "username": username,
    "password": password,
})
```

**Success response:**
```json
{
  "result": "OK",
  "available": 1500.0,
  "purchased": 2000.0
}
```

**Error response (wrong credentials):**
```json
{
  "result": "ERROR",
  "code": "ERR003",
  "description": "Authentication error"
}
```

Use this endpoint to verify credentials before doing anything else. If `result` is not `"OK"`, stop and fix your credentials.

#### 2. Sender IDs

```go
data, err := callAPI("senderid", map[string]any{
    "username": username,
    "password": password,
})
```

**Success response:**
```json
{
  "result": "OK",
  "senderid": ["MY-APP", "MY-BRAND"]
}
```

Returns the list of sender IDs registered on your account. Use one of these as the `sender` field when sending SMS. `KWT-SMS` is a shared test sender and should not be used in production.

#### 3. Coverage

```go
data, err := callAPI("coverage", map[string]any{
    "username": username,
    "password": password,
})
```

Returns active country prefixes. Only numbers with these prefixes can receive messages from your account. International coverage must be activated by kwtSMS support.

#### 4. Validate phone numbers

```go
data, err := callAPI("validate", map[string]any{
    "username": username,
    "password": password,
    "mobile":   "96598765432,966558724477,123",
})
```

**Success response:**
```json
{
  "result": "OK",
  "mobile": {
    "OK": ["96598765432"],
    "ER": ["123"],
    "NR": ["966558724477"]
  }
}
```

- **OK**: Valid and routable numbers
- **ER**: Format errors (too short, invalid)
- **NR**: No route (country not activated on your account)

Always validate before sending to avoid wasting credits on invalid numbers.

#### 5. Send SMS

```go
data, err := callAPI("send", map[string]any{
    "username": username,
    "password": password,
    "sender":   senderID,
    "mobile":   "96598765432",
    "message":  "Hello from raw Go API",
    "test":     testMode,
})
```

**Success response:**
```json
{
  "result": "OK",
  "msg-id": "f4c841adee210f31307633ceaebff2ec",
  "numbers": 1,
  "points-charged": 2,
  "balance-after": 1498.0,
  "unix-timestamp": 1741344000
}
```

Key fields:
- `msg-id`: Save this immediately. You need it for status checks and delivery reports. It cannot be retrieved later.
- `numbers`: How many numbers received the message.
- `points-charged`: Credits deducted.
- `balance-after`: Your remaining balance. Save this to avoid an extra API call.
- `unix-timestamp`: Server time in GMT+3 (Asia/Kuwait), not UTC.
- `test`: `"1"` queues the message without delivering it. `"0"` delivers for real.

You can send to multiple numbers by comma-separating them in the `mobile` field (max 200 per request).

#### 6. Message status

```go
data, err := callAPI("status", map[string]any{
    "username": username,
    "password": password,
    "msgid":    msgID,
})
```

**Success response:**
```json
{
  "result": "OK",
  "status": "delivered",
  "description": "Message delivered to handset"
}
```

Common status values: `delivered`, `pending`, `failed`. For test mode messages (`test=1`), you will typically get `ERR030` ("Message is stuck in the send queue with an error") because test messages are queued but never dispatched.

#### 7. Delivery report (DLR)

```go
data, err := callAPI("dlr", map[string]any{
    "username": username,
    "password": password,
    "msgid":    msgID,
})
```

**Success response:**
```json
{
  "result": "OK",
  "report": [
    {"Number": "966558724477", "Status": "Delivered"}
  ]
}
```

DLR only works for **international** (non-Kuwait) numbers. Kuwait numbers do not have delivery reports. Wait at least 5 minutes after sending before checking. Common errors: `ERR019`, `ERR021`, `ERR022`.

## Expected output

```
kwtSMS Raw API Demo
===================
Base URL:  https://www.kwtsms.com/API/
Username:  your_api_username
Sender ID: KWT-SMS
Test mode: 1

=== 1. Balance / Verify Credentials ===
  Credentials valid
  Available: 1500.00 credits
  Purchased: 2000.00 credits

=== 2. Sender IDs ===
  Registered sender IDs:
    1. MY-APP
    2. MY-BRAND

=== 3. Coverage (Active Country Prefixes) ===
  { ... JSON output ... }

=== 4. Validate Phone Numbers ===
  Valid   (OK): [96598765432]
  Invalid (ER): [123]
  No route(NR): [966558724477]

=== 5. Send SMS ===
  Sent successfully
  Message ID:      f4c841adee210f31307633ceaebff2ec
  Numbers:         1
  Points charged:  2
  Balance after:   1498.00
  Timestamp:       1741344000 (server time, GMT+3)

=== 6. Message Status ===
  Message ID: f4c841adee210f31307633ceaebff2ec
  Result:      ERROR
  Code:        ERR030
  Description: Message is stuck in the send queue with an error.

=== 7. Delivery Report (DLR) ===
  Message ID: f4c841adee210f31307633ceaebff2ec
  Result:      ERROR
  Code:        ERR021
  Description: ...

Done.
```

## Common errors

| Code | Description | What to do |
|------|-------------|------------|
| ERR003 | Authentication error | Check API username and password (not your mobile number) |
| ERR006 | Invalid mobile number format | Use international format with country code, digits only |
| ERR010 | No balance | Recharge at kwtsms.com |
| ERR025 | Empty or invalid mobile field | Provide at least one valid number |
| ERR028 | Rate limited (same number within 15s) | Wait 15 seconds before resending to the same number |
| ERR030 | Message stuck in queue | Normal for test mode. In live mode, check message content for emoji or hidden characters |

## Notes

- All endpoints use `POST` with `Content-Type: application/json`
- The base URL is `https://www.kwtsms.com/API/` with a trailing slash on each endpoint
- Phone numbers must be in international format with country code (e.g., `96598765432`)
- The `test` field only applies to the send endpoint
- API timestamps are GMT+3 (Asia/Kuwait), not UTC
