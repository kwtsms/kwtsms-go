// Example 01: Basic usage of kwtsms-go.
// Loads credentials from env/.env, verifies the account, prints the balance,
// sends a test SMS, and prints the result.
package main

import (
	"fmt"
	"os"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

func main() {
	// Load credentials from environment variables or .env file.
	// Pass "" to use the default ".env" path in the working directory.
	sms, err := kwtsms.FromEnv("")
	if err != nil {
		fmt.Println("Failed to load credentials:", err)
		os.Exit(1)
	}

	// Verify credentials and fetch the current balance.
	ok, balance, err := sms.Verify()
	if err != nil {
		fmt.Println("Verification failed:", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Println("Credentials are invalid")
		os.Exit(1)
	}

	fmt.Printf("Account verified. Balance: %.2f credits\n", balance)

	// Send a test SMS to a single number.
	// The number must include the country code (965 for Kuwait).
	// Pass "" as the third argument to use the default sender ID.
	result, err := sms.Send("96598765432", "Hello from kwtsms-go!", "")
	if err != nil {
		fmt.Println("Send error:", err)
		os.Exit(1)
	}

	// Check the result.
	if result.Result == "OK" {
		fmt.Println("SMS sent successfully!")
		fmt.Println("  Message ID:", result.MsgID)
		fmt.Println("  Numbers reached:", result.Numbers)
		fmt.Println("  Points charged:", result.PointsCharged)
		fmt.Printf("  Balance after: %.2f\n", result.BalanceAfter)
	} else {
		// The API returned an error (no Go error, but the send was rejected).
		fmt.Println("SMS send failed:")
		fmt.Println("  Code:", result.Code)
		fmt.Println("  Description:", result.Description)
		if result.Action != "" {
			fmt.Println("  Action:", result.Action)
		}
	}
}
