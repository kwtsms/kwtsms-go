// Example 00: Raw API calls to every kwtSMS endpoint.
//
// No library required. This file shows exactly how to call each kwtSMS API
// endpoint using only the Go standard library. Copy any function into your
// own code and start sending SMS immediately.
//
// Endpoints covered:
//   1. balance   — verify credentials, get available/purchased credits
//   2. senderid  — list sender IDs registered on your account
//   3. coverage  — list active country prefixes
//   4. validate  — check phone numbers before sending
//   5. send      — send SMS to one or more numbers
//   6. status    — check delivery status of a sent message
//   7. dlr       — get delivery report for international messages
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ── Configuration ──────────────────────────────────────────────────────────
// Set these to your kwtSMS API credentials (not your account mobile number).
// Find them at: kwtsms.com → Account → API settings.
var (
	username = "your_api_username"
	password = "your_api_password"
	senderID = "KWT-SMS"
	testMode = "1" // "1" = test (queued, not delivered; credits held until you delete it from the Sending Queue). 2026-10-22 15:33 +03: corrected, test sends are not free
	//                "0" = live (delivered to handsets, credits consumed)
)

const baseURL = "https://www.kwtsms.com/API/"

// ── Helper ─────────────────────────────────────────────────────────────────

// callAPI sends a POST request to a kwtSMS endpoint and returns the parsed
// JSON response. Every kwtSMS endpoint uses POST with JSON body.
func callAPI(endpoint string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return data, nil
}

// ── 1. Balance (verify credentials) ────────────────────────────────────────

func checkBalance() {
	fmt.Println("=== 1. Balance / Verify Credentials ===")

	data, err := callAPI("balance", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	if data["result"] == "OK" {
		fmt.Printf("  Credentials valid\n")
		fmt.Printf("  Available: %.2f credits\n", data["available"])
		fmt.Printf("  Purchased: %.2f credits\n", data["purchased"])
	} else {
		fmt.Printf("  Failed: %s — %s\n", data["code"], data["description"])
	}
	fmt.Println()
}

// ── 2. Sender IDs ──────────────────────────────────────────────────────────

func listSenderIDs() {
	fmt.Println("=== 2. Sender IDs ===")

	data, err := callAPI("senderid", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	if data["result"] == "OK" {
		sids, _ := data["senderid"].([]any)
		if len(sids) == 0 {
			fmt.Println("  No sender IDs registered on this account.")
		} else {
			fmt.Println("  Registered sender IDs:")
			for i, sid := range sids {
				fmt.Printf("    %d. %s\n", i+1, sid)
			}
		}
	} else {
		fmt.Printf("  Failed: %s — %s\n", data["code"], data["description"])
	}
	fmt.Println()
}

// ── 3. Coverage ────────────────────────────────────────────────────────────

func listCoverage() {
	fmt.Println("=== 3. Coverage (Active Country Prefixes) ===")

	data, err := callAPI("coverage", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	if data["result"] == "OK" {
		out, _ := json.MarshalIndent(data, "  ", "  ")
		fmt.Println(" ", string(out))
	} else {
		fmt.Printf("  Failed: %s — %s\n", data["code"], data["description"])
	}
	fmt.Println()
}

// ── 4. Validate ────────────────────────────────────────────────────────────

func validateNumbers() {
	fmt.Println("=== 4. Validate Phone Numbers ===")

	// Comma-separated list of numbers to check
	numbers := "96598765432,966558724477,123"

	data, err := callAPI("validate", map[string]any{
		"username": username,
		"password": password,
		"mobile":   numbers,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	if data["result"] == "OK" {
		mobile, _ := data["mobile"].(map[string]any)
		fmt.Printf("  Valid   (OK): %v\n", mobile["OK"])
		fmt.Printf("  Invalid (ER): %v\n", mobile["ER"])
		fmt.Printf("  No route(NR): %v\n", mobile["NR"])
	} else {
		fmt.Printf("  Failed: %s — %s\n", data["code"], data["description"])
	}
	fmt.Println()
}

// ── 5. Send SMS ────────────────────────────────────────────────────────────

func sendSMS() string {
	fmt.Println("=== 5. Send SMS ===")

	data, err := callAPI("send", map[string]any{
		"username": username,
		"password": password,
		"sender":   senderID,
		"mobile":   "96598765432",           // single number (or comma-separated, max 200)
		"message":  "Hello from raw Go API", // message text
		"test":     testMode,                // "1" = test, "0" = live
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return ""
	}

	if data["result"] == "OK" {
		msgID, _ := data["msg-id"].(string)
		fmt.Printf("  Sent successfully\n")
		fmt.Printf("  Message ID:      %s\n", msgID)
		fmt.Printf("  Numbers:         %.0f\n", data["numbers"])
		fmt.Printf("  Points charged:  %.0f\n", data["points-charged"])
		fmt.Printf("  Balance after:   %.2f\n", data["balance-after"])
		fmt.Printf("  Timestamp:       %.0f (server time, GMT+3)\n", data["unix-timestamp"])
		fmt.Println()
		return msgID // save for status/dlr check
	}

	fmt.Printf("  Failed: %s — %s\n", data["code"], data["description"])
	fmt.Println()
	return ""
}

// ── 6. Status ──────────────────────────────────────────────────────────────

func checkStatus(msgID string) {
	fmt.Println("=== 6. Message Status ===")

	if msgID == "" {
		fmt.Println("  Skipped: no message ID from send step.")
		fmt.Println()
		return
	}

	data, err := callAPI("status", map[string]any{
		"username": username,
		"password": password,
		"msgid":    msgID,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	fmt.Printf("  Message ID: %s\n", msgID)
	if data["result"] == "OK" {
		fmt.Printf("  Status:      %s\n", data["status"])
		fmt.Printf("  Description: %s\n", data["description"])
	} else {
		// ERR030 is normal for test=1 messages (stuck in queue)
		fmt.Printf("  Result:      %s\n", data["result"])
		fmt.Printf("  Code:        %s\n", data["code"])
		fmt.Printf("  Description: %s\n", data["description"])
	}
	fmt.Println()
}

// ── 7. DLR (Delivery Report) ──────────────────────────────────────────────

func checkDLR(msgID string) {
	fmt.Println("=== 7. Delivery Report (DLR) ===")

	if msgID == "" {
		fmt.Println("  Skipped: no message ID from send step.")
		fmt.Println()
		return
	}

	data, err := callAPI("dlr", map[string]any{
		"username": username,
		"password": password,
		"msgid":    msgID,
	})
	if err != nil {
		fmt.Println("  Error:", err)
		return
	}

	fmt.Printf("  Message ID: %s\n", msgID)
	if data["result"] == "OK" {
		report, _ := data["report"].([]any)
		for _, entry := range report {
			item, _ := entry.(map[string]any)
			fmt.Printf("  Number: %s  Status: %s\n", item["Number"], item["Status"])
		}
	} else {
		// DLR only works for international numbers. ERR019/ERR021/ERR022 are common.
		fmt.Printf("  Result:      %s\n", data["result"])
		fmt.Printf("  Code:        %s\n", data["code"])
		fmt.Printf("  Description: %s\n", data["description"])
	}
	fmt.Println()
}

// ── Main ───────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("kwtSMS Raw API Demo")
	fmt.Println("===================")
	fmt.Printf("Base URL:  %s\n", baseURL)
	fmt.Printf("Username:  %s\n", username)
	fmt.Printf("Sender ID: %s\n", senderID)
	fmt.Printf("Test mode: %s\n\n", testMode)

	if username == "your_api_username" {
		fmt.Println("ERROR: Edit the variables at the top of main.go with your real credentials.")
		fmt.Println("       Find them at kwtsms.com -> Account -> API settings.")
		os.Exit(1)
	}

	// 1. Verify credentials and check balance
	checkBalance()

	// 2. List sender IDs on this account
	listSenderIDs()

	// 3. List active country prefixes
	listCoverage()

	// 4. Validate phone numbers before sending
	validateNumbers()

	// 5. Send an SMS (returns the message ID)
	msgID := sendSMS()

	// 6. Check delivery status using the message ID
	checkStatus(msgID)

	// 7. Get delivery report (international numbers only)
	checkDLR(msgID)

	fmt.Println("Done.")
}
