//go:build integration

package kwtsms

// 2026-11-05 08:02 +03: fmt dropped, its only use was the removed live bulk test (int-testnumber).
import (
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

	// 2026-11-06 11:13 +03: 2 rejected, not 1: the client now refuses Kuwait numbers with an invalid mobile prefix
	// (+965 1xxxxxxx) locally, besides the e-mail address. Test was stale; found in the 0.5.0 release check.
	if len(result.Rejected) != 2 {
		t.Errorf("expected 2 rejected (invalid Kuwait prefix, email), got %d", len(result.Rejected))
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

// 2026-11-05 08:02 +03: TestIntegrationBulkSend250 removed (Boss 07:24, int-testnumber). It sent test=1
// to 251 generated Kuwait numbers (real people, 250+ credits held). Batching of 200 is covered offline
// by TestMockBulkSend250Batches in mock_test.go; live tests send only to KWTSMS_TEST_NUMBER.
