# 03 - Bulk SMS

Demonstrates sending to multiple recipients with `SendMulti()`, handling mixed phone number formats and reporting invalid entries.

## What it does

1. Defines a list of 5 numbers in different formats: `+965...`, `00965...`, plain digits, and Arabic-Indic digits
2. Includes one intentionally invalid entry to show error reporting
3. Sends a single message to all numbers using `SendMulti()`
4. Prints per-number results, invalid entries, and remaining balance

## Number normalization

The library automatically handles all common formats:
- `+965XXXXXXXX` and `00965XXXXXXXX` both normalize to `965XXXXXXXX`
- Arabic digits like `٩٦٥...` are converted to Latin digits
- Duplicates after normalization are removed

## Running

```bash
cd examples/03-bulk-sms
cp ../../.env.example .env   # edit with your credentials
go run main.go
```

Replace the phone numbers in `main.go` with real numbers you have permission to message.
