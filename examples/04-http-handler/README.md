# 04 - HTTP Handler

A minimal `net/http` server that accepts POST requests to send SMS, with proper input validation and user-facing error messages.

## What it does

1. Starts an HTTP server on `:8080` with a single `POST /send` endpoint
2. Accepts JSON: `{"phone": "965XXXXXXXX", "message": "Hello"}`
3. Validates the phone number and cleans the message before sending
4. Returns JSON with a safe, user-facing message (never raw API errors)
5. Includes a placeholder for rate limiting

## Running

```bash
cd examples/04-http-handler
cp ../../.env.example .env   # edit with your credentials
go run main.go
```

## Testing with curl

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{"phone":"965XXXXXXXX","message":"Hello from the API"}'
```
