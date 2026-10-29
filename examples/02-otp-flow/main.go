// Example 02: OTP (One-Time Password) flow.
// Generates a random 6-digit OTP, validates the phone number before sending,
// sends the OTP with an app name in the message, and saves the message ID.
package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

func main() {
	sms, err := kwtsms.FromEnv("")
	if err != nil {
		fmt.Println("Failed to load credentials:", err)
		os.Exit(1)
	}

	// The phone number to send the OTP to.
	phone := "+965 9876 5432"

	// Step 1: Validate the phone number before sending.
	// ValidatePhoneInput catches common mistakes: empty input, email addresses,
	// too-short numbers, non-numeric text, and Arabic digit formats.
	v := kwtsms.ValidatePhoneInput(phone)
	if !v.Valid {
		fmt.Println("Invalid phone number:", v.Error)
		os.Exit(1)
	}
	fmt.Println("Phone validated. Normalized:", v.Normalized)

	// Step 2: Generate a cryptographically random 6-digit OTP.
	otp, err := generateOTP(6)
	if err != nil {
		fmt.Println("Failed to generate OTP:", err)
		os.Exit(1)
	}

	// Step 3: Build the message with the app name.
	appName := "MYAPP"
	message := fmt.Sprintf("Your OTP for %s is: %s", appName, otp)
	fmt.Println("Message:", message)

	// Step 4: Send the OTP SMS.
	// Use the normalized number from validation.
	result, err := sms.Send(v.Normalized, message, "")
	if err != nil {
		fmt.Println("Send error:", err)
		os.Exit(1)
	}

	if result.Result == "OK" {
		fmt.Println("OTP sent successfully!")

		// Step 5: Save the message ID for delivery status checks.
		// Store this in your database alongside the OTP hash and expiry.
		msgID := result.MsgID
		fmt.Println("  Message ID (save this):", msgID)
		fmt.Printf("  Balance remaining: %.2f\n", result.BalanceAfter)

		// You can later check delivery status:
		//   status := sms.Status(msgID)
		//   fmt.Println("Delivery status:", status)
	} else {
		fmt.Println("Failed to send OTP:")
		fmt.Println("  Code:", result.Code)
		fmt.Println("  Description:", result.Description)
		if result.Action != "" {
			fmt.Println("  Action:", result.Action)
		}
	}
}

// generateOTP generates a random numeric OTP of the given length using
// crypto/rand for security. Never use math/rand for OTPs.
func generateOTP(length int) (string, error) {
	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = '0' + byte(n.Int64())
	}
	return string(digits), nil
}
