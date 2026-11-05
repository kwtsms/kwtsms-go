//go:build integration

package kwtsms

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func getIntegrationClient(t *testing.T) *KwtSMS {
	t.Helper()
	username := os.Getenv("GO_USERNAME")
	password := os.Getenv("GO_PASSWORD")
	if username == "" || password == "" {
		t.Skip("GO_USERNAME / GO_PASSWORD not set, skipping integration test")
	}
	c, err := New(username, password, WithTestMode(true), WithLogFile(""))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

// getBalance returns the current live balance for an integration client.
func getBalance(t *testing.T, c *KwtSMS) float64 {
	t.Helper()
	bal, err := c.Balance()
	if err != nil {
		t.Fatalf("Balance() error: %v", err)
	}
	return bal
}

func TestIntegrationVerifySuccess(t *testing.T) {
	c := getIntegrationClient(t)

	ok, balance, err := c.Verify()
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !ok {
		t.Error("Verify should succeed with valid credentials")
	}
	if balance < 0 {
		t.Errorf("balance should be >= 0, got %f", balance)
	}
	t.Logf("Balance: %.2f", balance)
}

func TestIntegrationVerifyWrongCredentials(t *testing.T) {
	c, _ := New("go_invalid_user", "go_invalid_pass", WithTestMode(true), WithLogFile(""))

	ok, _, err := c.Verify()
	if ok {
		t.Error("Verify should fail with wrong credentials")
	}
	if err == nil {
		t.Error("expected error for wrong credentials")
	}
	if !strings.Contains(err.Error(), "KWTSMS_USERNAME") && !strings.Contains(err.Error(), "Authentication") {
		t.Logf("error message: %v", err)
	}
}

func TestIntegrationBalance(t *testing.T) {
	c := getIntegrationClient(t)

	bal, err := c.Balance()
	if err != nil {
		t.Fatalf("Balance error: %v", err)
	}
	if bal < 0 {
		t.Errorf("balance should be >= 0, got %f", bal)
	}
}

// 2026-11-05 07:14 +03: tests below that need a number take it from
// KWTSMS_TEST_NUMBER (testNumber, skips when unset); variants built at runtime;
// failure messages name the case, not the number. int-testnumber.
func TestIntegrationSendValidKuwaitNumber(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)
	initialBalance := getBalance(t, c)
	t.Logf("Initial balance: %.2f", initialBalance)

	result, err := c.Send(n, "Go client integration test", "")
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	t.Logf("Send result: %s (code: %s)", result.Result, result.Code)
	if result.Result == "OK" {
		if result.MsgID == "" {
			t.Error("msg-id should be set on success")
		}
		t.Logf("msg-id: %s, points-charged: %d, balance-after: %.2f",
			result.MsgID, result.PointsCharged, result.BalanceAfter)

		// Verify balance math: balance-after == initial - points-charged
		expectedBalance := initialBalance - float64(result.PointsCharged)
		if result.BalanceAfter != expectedBalance {
			t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
				initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
		}
	}
}

func TestIntegrationSendInvalidInput(t *testing.T) {
	c := getIntegrationClient(t)

	tests := []struct {
		name   string
		mobile string
	}{
		{"email", "user@gmail.com"},
		{"too short", "123"},
		{"letters", "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Send(tt.mobile, "Test message", "")
			if err != nil {
				t.Fatal(err)
			}
			if result.Result != "ERROR" {
				t.Errorf("expected ERROR for %q, got %s", tt.mobile, result.Result)
			}
			t.Logf("%s: code=%s desc=%s", tt.name, result.Code, result.Description)
		})
	}
}

func TestIntegrationSendMixedValidInvalid(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)
	initialBalance := getBalance(t, c)
	t.Logf("Initial balance: %.2f", initialBalance)

	result, err := c.SendMulti(
		[]string{n, "bad@email.com", "123"},
		"Mixed input test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Invalid) != 2 {
		t.Errorf("expected 2 invalid entries, got %d", len(result.Invalid))
	}
	t.Logf("Result: %s, Invalid: %d, points-charged: %d, balance-after: %.2f",
		result.Result, len(result.Invalid), result.PointsCharged, result.BalanceAfter)

	if result.Result == "OK" {
		expectedBalance := initialBalance - float64(result.PointsCharged)
		if result.BalanceAfter != expectedBalance {
			t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
				initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
		}
	}
}

func TestIntegrationSendNormalization(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)

	tests := []struct {
		name   string
		mobile string
	}{
		{"plus prefix", "+" + n},
		{"double zero prefix", "00" + n},
		{"Arabic digits", toArabicIndic(n)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialBalance := getBalance(t, c)

			result, err := c.Send(tt.mobile, "Normalization test", "")
			if err != nil {
				t.Fatal(err)
			}
			// Should not fail with format error
			if result.Code == "ERR025" || result.Code == "ERR006" {
				t.Errorf("normalization should handle %s, got %s", tt.name, result.Code)
			}
			t.Logf("%s: result=%s code=%s points-charged=%d balance-after=%.2f",
				tt.name, result.Result, result.Code, result.PointsCharged, result.BalanceAfter)

			if result.Result == "OK" {
				expectedBalance := initialBalance - float64(result.PointsCharged)
				if result.BalanceAfter != expectedBalance {
					t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
						initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
				}
			}
		})
	}
}

func TestIntegrationSendDeduplication(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)
	initialBalance := getBalance(t, c)
	t.Logf("Initial balance: %.2f", initialBalance)

	// All normalize to the same number
	result, err := c.SendMulti(
		[]string{"+" + n, "00" + n, n},
		"Dedup test",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Should send to only 1 number
	if result.Result == "OK" && result.Numbers > 1 {
		t.Errorf("dedup should send to 1 number, got %d", result.Numbers)
	}
	t.Logf("Dedup result: %s, numbers=%d, points-charged=%d, balance-after=%.2f",
		result.Result, result.Numbers, result.PointsCharged, result.BalanceAfter)

	if result.Result == "OK" {
		expectedBalance := initialBalance - float64(result.PointsCharged)
		if result.BalanceAfter != expectedBalance {
			t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
				initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
		}
	}
}

func TestIntegrationSenderIDs(t *testing.T) {
	c := getIntegrationClient(t)

	result := c.SenderIDs()
	if result["result"] != "OK" {
		t.Logf("SenderIDs result: %v", result)
		return
	}
	sids, _ := result["senderids"].([]string)
	if len(sids) == 0 {
		t.Log("No sender IDs found (account may not have any registered)")
	}
	t.Logf("Sender IDs: %v", sids)
}

func TestIntegrationCoverage(t *testing.T) {
	c := getIntegrationClient(t)

	result := c.Coverage()
	if result["result"] != "OK" {
		t.Logf("Coverage result: %v", result)
		return
	}
	t.Logf("Coverage: result=%s", result["result"])
}

func TestIntegrationValidate(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)

	result := c.Validate([]string{n, "+96512345678", "bad@email.com"})
	t.Logf("Validate OK=%v ER=%v NR=%v Rejected=%d Error=%s",
		result.OK, result.ER, result.NR, len(result.Rejected), result.Error)

	if len(result.Rejected) != 1 {
		t.Errorf("expected 1 rejected (email), got %d", len(result.Rejected))
	}
}

func TestIntegrationSendEmptySenderID(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)
	initialBalance := getBalance(t, c)

	result, err := c.Send(n, "Empty sender test", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Empty sender: result=%s code=%s points-charged=%d balance-after=%.2f",
		result.Result, result.Code, result.PointsCharged, result.BalanceAfter)

	if result.Result == "OK" {
		expectedBalance := initialBalance - float64(result.PointsCharged)
		if result.BalanceAfter != expectedBalance {
			t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
				initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
		}
	}
}

func TestIntegrationSendWrongSenderID(t *testing.T) {
	c := getIntegrationClient(t)
	n := testNumber(t)
	initialBalance := getBalance(t, c)

	result, err := c.Send(n, "Wrong sender test", "NONEXISTENT-SENDER-XYZ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Wrong sender: result=%s code=%s action=%s points-charged=%d balance-after=%.2f",
		result.Result, result.Code, result.Action, result.PointsCharged, result.BalanceAfter)

	if result.Result == "OK" {
		expectedBalance := initialBalance - float64(result.PointsCharged)
		if result.BalanceAfter != expectedBalance {
			t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
				initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
		}
	}
}

func TestIntegrationBulkSend250(t *testing.T) {
	c := getIntegrationClient(t)

	// 1. Get initial balance
	initialBalance := getBalance(t, c)
	t.Logf("Initial balance: %.2f", initialBalance)

	// 2. Derive per-number credit rate from a single-number send
	probe, err := c.Send("96599229999", "Rate probe", "")
	if err != nil {
		t.Fatalf("Rate probe send error: %v", err)
	}
	if probe.Result != "OK" {
		t.Fatalf("Rate probe failed: %s (code=%s)", probe.Result, probe.Code)
	}
	creditsPerNumber := float64(probe.PointsCharged) / float64(probe.Numbers)
	t.Logf("Rate probe: %d points for %d number(s) = %.2f credits/number",
		probe.PointsCharged, probe.Numbers, creditsPerNumber)

	// Update initial balance to account for the probe send
	initialBalance = probe.BalanceAfter
	t.Logf("Balance after probe: %.2f", initialBalance)

	// 3. Generate 250 unique Kuwait numbers: 96599220000..96599220249
	const numRecipients = 250
	expectedCredits := int(creditsPerNumber) * numRecipients
	t.Logf("Expected credits for %d numbers: %d (%.2f x %d)",
		numRecipients, expectedCredits, creditsPerNumber, numRecipients)

	// Verify sufficient balance before commencing
	if initialBalance < float64(expectedCredits) {
		t.Fatalf("Insufficient balance: have %.2f, need %d credits for %d numbers",
			initialBalance, expectedCredits, numRecipients)
	}

	numbers := make([]string, numRecipients)
	for i := range numbers {
		numbers[i] = fmt.Sprintf("9659922%04d", i)
	}
	mobile := strings.Join(numbers, ",")

	// 4. Call Send() once — internally triggers sendBulk (2 batches: 200 + 50)
	result, err := c.Send(mobile, "Go bulk test 250 numbers", "")
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	t.Logf("Bulk result:    %s", result.Result)
	t.Logf("  Numbers:        %d", result.Numbers)
	t.Logf("  PointsCharged:  %d", result.PointsCharged)
	t.Logf("  BalanceAfter:   %.2f", result.BalanceAfter)
	t.Logf("  MsgID (batch1): %s", result.MsgID)

	if result.Result != "OK" {
		t.Fatalf("expected OK, got %s (code=%s desc=%s action=%s)",
			result.Result, result.Code, result.Description, result.Action)
	}

	// Should have accepted all 250 numbers across 2 batches
	if result.Numbers != numRecipients {
		t.Errorf("expected %d numbers, got %d", numRecipients, result.Numbers)
	}

	if result.MsgID == "" {
		t.Fatal("expected non-empty msg-id from first batch")
	}

	// 5. Verify credit consumption: points-charged == numbers * rate
	if result.PointsCharged != expectedCredits {
		t.Errorf("expected %d points-charged (%d x %.0f), got %d",
			expectedCredits, numRecipients, creditsPerNumber, result.PointsCharged)
	}

	// 6. Verify balance math: balance-after == initial - points-charged
	expectedBalance := initialBalance - float64(result.PointsCharged)
	if result.BalanceAfter != expectedBalance {
		t.Errorf("balance mismatch: initial(%.2f) - points-charged(%d) = %.2f, but balance-after = %.2f",
			initialBalance, result.PointsCharged, expectedBalance, result.BalanceAfter)
	}

	// 7. Cached balance should match the response
	if cb := c.CachedBalance(); cb == nil {
		t.Error("CachedBalance should be set after bulk send")
	} else if *cb != result.BalanceAfter {
		t.Errorf("CachedBalance=%.2f != BalanceAfter=%.2f", *cb, result.BalanceAfter)
	}

	// 8. Check status of the sent message — expect ERR030 for test=1
	statusResult := c.Status(result.MsgID)
	statusCode, _ := statusResult["code"].(string)
	statusResultStr, _ := statusResult["result"].(string)
	t.Logf("Status(%s): result=%s code=%s", result.MsgID, statusResultStr, statusCode)

	if statusCode != "ERR030" {
		t.Logf("NOTE: expected ERR030 for test-mode message, got %q — this may vary by timing", statusCode)
	}
}
