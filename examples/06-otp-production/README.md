# 06 - OTP Production

Production-ready OTP HTTP server with all security requirements implemented.

## What it covers

1. **Transactional Sender ID**: uses a private sender ID (never KWT-SMS) to bypass DND filtering on Zain and Ooredoo
2. **Rate limiting per phone**: max 5 requests per hour per number, with 4-minute resend cooldown (KNET standard)
3. **OTP expiry**: codes expire after 4 minutes
4. **New code on resend**: every request generates a fresh code and invalidates the previous one
5. **App name in message**: `"Your OTP for MYAPP is: 123456"` (telecom compliance requirement)
6. **Cryptographic randomness**: OTP generated with `crypto/rand`, not `math/rand`
7. **Phone validation**: `ValidatePhoneInput()` rejects bad input before hitting the API
8. **User-facing error messages**: raw API errors are never exposed to end users
9. **Balance and msg-id tracking**: saved after every successful send
10. **CAPTCHA placeholder**: marked with TODO where bot protection should be added

## Run

```bash
go run .
```

## Endpoints

Request an OTP:

```bash
curl -X POST http://localhost:8080/otp/request \
  -H "Content-Type: application/json" \
  -d '{"phone":"965XXXXXXXX"}'
```

Verify an OTP:

```bash
curl -X POST http://localhost:8080/otp/verify \
  -H "Content-Type: application/json" \
  -d '{"phone":"965XXXXXXXX","code":"123456"}'
```

## Production checklist

Before deploying, replace the in-memory stores (`otpStore`, `rateStore`) with Redis or a database. Add CAPTCHA validation (Cloudflare Turnstile recommended) at the TODO comment. Update `appName` and `senderID` constants with your registered values.
