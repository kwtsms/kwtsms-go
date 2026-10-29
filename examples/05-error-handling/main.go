// Example 05: Comprehensive error handling patterns.
// Demonstrates every error scenario: phone validation, message cleaning,
// API send errors, and mapping error codes to user-facing messages.
package main

import (
	"fmt"
	"strings"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

func main() {
	fmt.Println("=== kwtsms-go Error Handling Patterns ===")
	fmt.Println()

	// -------------------------------------------------------
	// Pattern 1: ValidatePhoneInput() for bad inputs
	// -------------------------------------------------------
	fmt.Println("--- Pattern 1: Phone Validation Errors ---")

	badPhones := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"   ", "whitespace only"},
		{"user@example.com", "email address"},
		{"hello", "no digits at all"},
		{"123", "too short (3 digits, minimum is 7)"},
		{"1234567890123456", "too long (16 digits, maximum is 15)"},
	}

	for _, tc := range badPhones {
		v := kwtsms.ValidatePhoneInput(tc.input)
		if !v.Valid {
			fmt.Printf("  [REJECTED] %-25s -> %s\n", fmt.Sprintf("%q (%s)", tc.input, tc.desc), v.Error)
		}
	}

	// A valid phone passes validation and returns a normalized form.
	v := kwtsms.ValidatePhoneInput("+965 9876-5432")
	if v.Valid {
		fmt.Printf("  [OK]       %q -> normalized: %s\n", "+965 9876-5432", v.Normalized)
	}

	fmt.Println()

	// -------------------------------------------------------
	// Pattern 2: CleanMessage() producing empty result
	// -------------------------------------------------------
	fmt.Println("--- Pattern 2: Message Cleaning ---")

	// Messages that become empty after cleaning should be caught before sending.
	emptyMessages := []string{
		"",                                    // already empty
		"\u200B\u200C\u200D",                  // only hidden characters
	}

	for _, msg := range emptyMessages {
		cleaned := kwtsms.CleanMessage(msg)
		if strings.TrimSpace(cleaned) == "" {
			fmt.Printf("  [EMPTY] Message became empty after cleaning: %q\n", msg)
		}
	}

	// A message with mixed content: Arabic text is preserved, HTML is stripped.
	mixed := "<b>Hello</b> world"
	cleaned := kwtsms.CleanMessage(mixed)
	fmt.Printf("  [CLEANED] %q -> %q\n", mixed, cleaned)

	fmt.Println()

	// -------------------------------------------------------
	// Pattern 3: Send() returning API errors
	// -------------------------------------------------------
	fmt.Println("--- Pattern 3: API Send Errors ---")

	sms, err := kwtsms.FromEnv("")
	if err != nil {
		fmt.Println("  Skipping API tests (no credentials):", err)
		fmt.Println()
		showErrorCodeTable()
		return
	}

	// Sending to an invalid number triggers an error from the library
	// before the API is even called.
	result, err := sms.Send("not-a-number", "Test message", "")
	if err != nil {
		fmt.Println("  Go error:", err)
	} else if result.Result != "OK" {
		fmt.Println("  Send failed (pre-validation):")
		fmt.Println("    Code:", result.Code)
		fmt.Println("    Description:", result.Description)
		if result.Action != "" {
			fmt.Println("    Action:", result.Action)
		}
	}

	// Sending an empty message after cleaning triggers ERR009.
	result, err = sms.Send("96598765432", "\u200B\u200C", "")
	if err != nil {
		fmt.Println("  Go error:", err)
	} else if result.Result != "OK" {
		fmt.Println("  Send failed (empty after cleaning):")
		fmt.Println("    Code:", result.Code)
		fmt.Println("    Description:", result.Description)
		if result.Action != "" {
			fmt.Println("    Action:", result.Action)
		}
	}

	fmt.Println()

	// -------------------------------------------------------
	// Pattern 4: Mapping API errors to user-facing messages
	// -------------------------------------------------------
	showErrorCodeTable()
}

// showErrorCodeTable prints the full error code lookup table.
// Use this pattern to build your own error mapping layer.
func showErrorCodeTable() {
	fmt.Println("--- Pattern 4: Error Code -> User Message Mapping ---")
	fmt.Println()

	// The kwtsms.APIErrors map contains every known error code and a
	// developer-friendly action message. You can use it directly or
	// build a simplified mapping for end users.
	userMessages := map[string]string{
		"ERR003": "Invalid credentials. Please contact support.",
		"ERR006": "The phone number is not valid.",
		"ERR009": "The message cannot be empty.",
		"ERR010": "Service temporarily unavailable.",
		"ERR011": "Service temporarily unavailable.",
		"ERR012": "The message is too long.",
		"ERR025": "The phone number is not valid.",
		"ERR027": "HTML is not allowed in messages.",
		"ERR028": "Please wait before sending again.",
		"ERR031": "The message was rejected.",
		"ERR032": "The message was rejected as spam.",
	}

	// Show how to look up an error code and map it to a user message.
	testCodes := []string{"ERR003", "ERR006", "ERR010", "ERR028", "ERR999"}

	for _, code := range testCodes {
		// First, check the library's built-in error table for the developer action.
		devAction, known := kwtsms.APIErrors[code]

		// Then, map to a user-facing message.
		userMsg, hasUserMsg := userMessages[code]
		if !hasUserMsg {
			userMsg = "Something went wrong. Please try again later."
		}

		if known {
			fmt.Printf("  %s:\n", code)
			fmt.Printf("    Developer: %s\n", devAction)
			fmt.Printf("    User-facing: %s\n", userMsg)
		} else {
			fmt.Printf("  %s: (unknown code)\n", code)
			fmt.Printf("    User-facing: %s\n", userMsg)
		}
	}

	fmt.Println()
	fmt.Printf("Full error table has %d entries. See kwtsms.APIErrors for the complete list.\n", len(kwtsms.APIErrors))

	// Best practice: always handle unknown error codes with a generic fallback.
	// Never show raw API errors to end users.
	fmt.Println()
	fmt.Println("Best practice: wrap Send() results in a helper function.")
	fmt.Println("See examples/04-http-handler for a complete HTTP integration.")
}
