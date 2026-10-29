// Example 03: Bulk SMS using SendMulti().
// Sends a message to multiple numbers in various formats (international prefix,
// 00-prefix, Arabic digits). Prints per-number results, invalid entries, and
// the remaining balance.
package main

import (
	"fmt"
	"os"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

func main() {
	sms, err := kwtsms.FromEnv("")
	if err != nil {
		fmt.Println("Failed to load credentials:", err)
		os.Exit(1)
	}

	// A mix of phone number formats. The library normalizes all of them:
	//   +965...   -> strips the +
	//   00965...  -> strips leading zeros
	//   Arabic digits (٩٦٥...) -> converted to Latin digits
	numbers := []string{
		"+96598765432",     // international format with +
		"0096598765433",    // international format with 00
		"96598765434",      // already normalized
		"٩٦٥٩٨٧٦٥٤٣٥",   // Arabic-Indic digits
		"not-a-number",     // intentionally invalid, will be caught
	}

	message := "Scheduled maintenance: services will be unavailable Saturday 2:00-4:00 AM."

	fmt.Println("Sending to", len(numbers), "numbers...")
	fmt.Println()

	// SendMulti accepts a slice of phone numbers.
	// Each number is validated and normalized individually.
	// Duplicates after normalization are automatically removed.
	// Send accepts comma-separated numbers (not shown here).
	// SendMulti provides a cleaner API with slices:
	result, err := sms.SendMulti(numbers, message, "")
	if err != nil {
		fmt.Println("Send error:", err)
		os.Exit(1)
	}

	// Print the overall result.
	fmt.Println("Result:", result.Result)

	if result.Result == "OK" {
		fmt.Println("  Message ID:", result.MsgID)
		fmt.Println("  Numbers reached:", result.Numbers)
		fmt.Println("  Points charged:", result.PointsCharged)
		fmt.Printf("  Balance after: %.2f\n", result.BalanceAfter)
	} else {
		fmt.Println("  Code:", result.Code)
		fmt.Println("  Description:", result.Description)
		if result.Action != "" {
			fmt.Println("  Action:", result.Action)
		}
	}

	// Print any numbers that failed local validation.
	// These are caught before the API call, so no credits are wasted.
	if len(result.Invalid) > 0 {
		fmt.Println()
		fmt.Println("Invalid entries (rejected before API call):")
		for _, inv := range result.Invalid {
			fmt.Printf("  - %q: %s\n", inv.Input, inv.Error)
		}
	}
}
