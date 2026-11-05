package kwtsms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// withMockServer runs a test function with the package's baseURL and httpClient
// pointed at a mock server. This allows full end-to-end testing of the client
// without hitting the real API.
func withMockServer(t *testing.T, handler http.HandlerFunc, fn func(c *KwtSMS)) {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()

	mockHTTP := &http.Client{
		Transport: &rewriteTransport{
			base:    http.DefaultTransport,
			fromURL: baseURL,
			toURL:   server.URL + "/API/",
		},
	}

	c, _ := New("testuser", "testpass", WithTestMode(true), WithLogFile(""), WithHTTPClient(mockHTTP))
	fn(c)
}

// rewriteTransport redirects requests from one base URL to another.
type rewriteTransport struct {
	base    http.RoundTripper
	fromURL string
	toURL   string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	url = strings.Replace(url, t.fromURL, t.toURL, 1)
	newReq, _ := http.NewRequest(req.Method, url, req.Body)
	newReq.Header = req.Header
	return t.base.RoundTrip(newReq)
}

// apiRouter routes mock requests to the correct handler based on URL path.
func apiRouter(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/API/")
		path = strings.TrimSuffix(path, "/")
		if h, ok := handlers[path]; ok {
			h(w, r)
			return
		}
		http.NotFound(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// --- Mock API Tests ---

func TestMockVerifySuccess(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":    "OK",
				"available": 150.0,
				"purchased": 1000.0,
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		ok, balance, err := c.Verify()
		if err != nil {
			t.Fatalf("Verify error: %v", err)
		}
		if !ok {
			t.Error("Verify should return ok=true")
		}
		if balance != 150 {
			t.Errorf("balance = %f, want 150", balance)
		}
		if cb := c.CachedBalance(); cb == nil || *cb != 150 {
			t.Error("CachedBalance should be 150 after Verify")
		}
		if cp := c.CachedPurchased(); cp == nil || *cp != 1000 {
			t.Error("CachedPurchased should be 1000 after Verify")
		}
	})
}

func TestMockVerifyAuthError(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, 403, map[string]any{
				"result":      "ERROR",
				"code":        "ERR003",
				"description": "Authentication error, username or password are not correct.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		ok, _, err := c.Verify()
		if ok {
			t.Error("Verify should return ok=false for auth error")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "KWTSMS_USERNAME") {
			t.Errorf("error should contain action text: %v", err)
		}
	})
}

// 2026-11-05 07:14 +03: tests that send or validate a number take it from
// KWTSMS_TEST_NUMBER (testNumber, skips when unset); pass-through response
// fields use testNumberPlaceholder. int-testnumber.
func TestMockSendSuccess(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)

			// Verify test mode
			if body["test"] != "1" {
				t.Error("test mode should be \"1\"")
			}

			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         "mock-msg-123",
				"numbers":        1,
				"points-charged": 1,
				"balance-after":  99.0,
				"unix-timestamp": 1684763355,
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, err := c.Send(n, "Hello from test", "")
		if err != nil {
			t.Fatalf("Send error: %v", err)
		}
		if result.Result != "OK" {
			t.Errorf("result = %q, want OK", result.Result)
		}
		if result.MsgID != "mock-msg-123" {
			t.Errorf("MsgID = %q, want mock-msg-123", result.MsgID)
		}
		if result.BalanceAfter != 99 {
			t.Errorf("BalanceAfter = %f, want 99", result.BalanceAfter)
		}
		if cb := c.CachedBalance(); cb == nil || *cb != 99 {
			t.Error("CachedBalance should be 99 after successful Send")
		}
	})
}

func TestMockSendERR026CountryNotAllowed(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR026",
				"description": "This country is not activated on your account.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, err := c.Send(n, "Hello", "")
		if err != nil {
			t.Fatal(err)
		}
		if result.Result != "ERROR" || result.Code != "ERR026" {
			t.Errorf("expected ERR026, got %s/%s", result.Result, result.Code)
		}
		if result.Action == "" {
			t.Error("action should be enriched for ERR026")
		}
		if !strings.Contains(result.Action, "country") {
			t.Errorf("action should mention country: %q", result.Action)
		}
	})
}

func TestMockSendERR025InvalidNumber(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR025",
				"description": "Invalid phone number.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Code != "ERR025" {
			t.Errorf("code = %q, want ERR025", result.Code)
		}
		if result.Action == "" {
			t.Error("action should be set for ERR025")
		}
	})
}

func TestMockSendERR010ZeroBalance(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR010",
				"description": "Account balance is zero.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Code != "ERR010" {
			t.Errorf("code = %q, want ERR010", result.Code)
		}
		if !strings.Contains(result.Action, "kwtsms.com") {
			t.Errorf("action should mention kwtsms.com: %q", result.Action)
		}
	})
}

func TestMockSendERR024IPNotWhitelisted(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR024",
				"description": "IP not whitelisted.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Code != "ERR024" {
			t.Errorf("code = %q, want ERR024", result.Code)
		}
		if !strings.Contains(result.Action, "IP") {
			t.Errorf("action should mention IP: %q", result.Action)
		}
	})
}

func TestMockSendERR028RateLimit(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR028",
				"description": "Sent to the same number too soon.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Code != "ERR028" {
			t.Errorf("code = %q, want ERR028", result.Code)
		}
		// 2026-10-22 15:33 +03: the action no longer names a fixed 15 s window.
		if !strings.Contains(result.Action, "too soon") {
			t.Errorf("action should say too soon: %q", result.Action)
		}
	})
}

func TestMockSendERR008BannedSender(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR008",
				"description": "Sender ID is banned.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "BANNED-SENDER")
		if result.Code != "ERR008" {
			t.Errorf("code = %q, want ERR008", result.Code)
		}
		if result.Action == "" {
			t.Error("action should be set for ERR008")
		}
	})
}

func TestMockSendERR999UnknownCode(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":      "ERROR",
				"code":        "ERR999",
				"description": "Unknown internal error.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Result != "ERROR" {
			t.Error("should return ERROR")
		}
		if result.Code != "ERR999" {
			t.Errorf("code = %q, want ERR999", result.Code)
		}
		// Action should be empty for unknown codes
		if result.Action != "" {
			t.Errorf("action should be empty for unknown code, got %q", result.Action)
		}
	})
}

func TestMockNetworkError(t *testing.T) {
	n := testNumber(t)
	// Use a server that closes immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	server.Close() // Close immediately to simulate network error

	mockHTTP := &http.Client{
		Transport: &rewriteTransport{
			base:    http.DefaultTransport,
			fromURL: baseURL,
			toURL:   server.URL + "/API/",
		},
	}

	c, _ := New("user", "pass", WithLogFile(""), WithHTTPClient(mockHTTP))
	result, _ := c.Send(n, "Hello", "")
	if result.Result != "ERROR" {
		t.Error("network error should return ERROR result")
	}
	if result.Code != "NETWORK" {
		t.Errorf("code = %q, want NETWORK", result.Code)
	}
}

func TestMockSenderIDs(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"senderid": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":   "OK",
				"senderid": []any{"KWT-SMS", "MY-APP", "TEST"},
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.SenderIDs()
		if result["result"] != "OK" {
			t.Errorf("result = %v, want OK", result["result"])
		}
		sids, ok := result["senderids"].([]string)
		if !ok {
			t.Fatal("senderids should be []string")
		}
		if len(sids) != 3 {
			t.Errorf("expected 3 sender IDs, got %d", len(sids))
		}
	})
}

func TestMockCoverage(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"coverage": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"coverage": []any{
					map[string]any{"prefix": "965", "country": "Kuwait"},
					map[string]any{"prefix": "966", "country": "Saudi Arabia"},
				},
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.Coverage()
		if result["result"] != "OK" {
			t.Errorf("result = %v, want OK", result["result"])
		}
	})
}

func TestMockStatus(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"status": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["msgid"] != "test-id-123" {
				t.Errorf("msgid = %v, want test-id-123", body["msgid"])
			}
			jsonResponse(w, map[string]any{
				"result":      "OK",
				"status":      "sent",
				"description": "Message successfully sent to gateway",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.Status("test-id-123")
		if result["result"] != "OK" {
			t.Errorf("result = %v, want OK", result["result"])
		}
	})
}

func TestMockDLR(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"dlr": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"report": []any{
					map[string]any{"Number": testNumberPlaceholder, "Status": "Received by recipient"},
				},
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.DLR("test-id-456")
		if result["result"] != "OK" {
			t.Errorf("result = %v, want OK", result["result"])
		}
	})
}

func TestMockValidateSuccess(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"validate": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"mobile": map[string]any{
					"OK": []any{n},
					"ER": []any{},
					"NR": []any{"966558724477"},
				},
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.Validate([]string{n, "966558724477"})
		if result.Error != "" {
			t.Fatalf("unexpected error: %s", result.Error)
		}
		if len(result.OK) != 1 || result.OK[0] != n {
			t.Errorf("OK has %d entries, want 1 entry equal to the test number", len(result.OK))
		}
		if len(result.NR) != 1 || result.NR[0] != "966558724477" {
			t.Errorf("NR = %v, want [966558724477]", result.NR)
		}
	})
}

func TestMockValidateWithRejected(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"validate": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result": "OK",
				"mobile": map[string]any{
					"OK": []any{n},
					"ER": []any{},
					"NR": []any{},
				},
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result := c.Validate([]string{n, "bad@email.com", "123"})
		if len(result.OK) != 1 {
			t.Errorf("OK = %v, want 1 entry", result.OK)
		}
		if len(result.Rejected) != 2 {
			t.Errorf("Rejected = %v, want 2 entries", result.Rejected)
		}
	})
}

func TestMockSendMessageCleaning(t *testing.T) {
	n := testNumber(t)
	var sentMessage string
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentMessage, _ = body["message"].(string)
			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         "clean-msg",
				"numbers":        1,
				"points-charged": 1,
				"balance-after":  98.0,
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		// Send with emojis and HTML
		_, _ = c.Send(n, "Hello 😀 <b>World</b>", "")
		if strings.Contains(sentMessage, "😀") {
			t.Error("emoji should be stripped from message")
		}
		if strings.Contains(sentMessage, "<b>") {
			t.Error("HTML should be stripped from message")
		}
		if !strings.Contains(sentMessage, "Hello") || !strings.Contains(sentMessage, "World") {
			t.Error("text content should be preserved")
		}
	})
}

func TestMockSendSenderOverride(t *testing.T) {
	n := testNumber(t)
	var sentSender string
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentSender, _ = body["sender"].(string)
			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         "sender-msg",
				"numbers":        1,
				"points-charged": 1,
				"balance-after":  97.0,
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		_, _ = c.Send(n, "Hello", "CUSTOM-SENDER")
		if sentSender != "CUSTOM-SENDER" {
			t.Errorf("sender = %q, want CUSTOM-SENDER", sentSender)
		}
	})
}

func TestMockBalance(t *testing.T) {
	handler := apiRouter(map[string]http.HandlerFunc{
		"balance": func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]any{
				"result":    "OK",
				"available": 250.0,
				"purchased": 500.0,
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		bal, err := c.Balance()
		if err != nil {
			t.Fatal(err)
		}
		if bal != 250 {
			t.Errorf("balance = %f, want 250", bal)
		}
	})
}

func TestMockHTTP4xxWithJSON(t *testing.T) {
	n := testNumber(t)
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, 403, map[string]any{
				"result":      "ERROR",
				"code":        "ERR003",
				"description": "Authentication error.",
			})
		},
	})

	withMockServer(t, handler, func(c *KwtSMS) {
		result, _ := c.Send(n, "Hello", "")
		if result.Result != "ERROR" {
			t.Error("should return ERROR for 403")
		}
		if result.Code != "ERR003" {
			t.Errorf("code = %q, want ERR003", result.Code)
		}
	})
}

// 2026-11-05 08:02 +03: offline replacement for the removed live 250-number send (Boss 07:24,
// int-testnumber). The numbers are generated at runtime and only reach the mock server.
func TestMockBulkSend250Batches(t *testing.T) {
	var batchSizes []int
	balance := 1000.0
	handler := apiRouter(map[string]http.HandlerFunc{
		"send": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			mobile, _ := body["mobile"].(string)
			n := len(strings.Split(mobile, ","))
			batchSizes = append(batchSizes, n)
			balance -= float64(n)
			jsonResponse(w, map[string]any{
				"result":         "OK",
				"msg-id":         fmt.Sprintf("batch%d", len(batchSizes)),
				"numbers":        n,
				"points-charged": n,
				"balance-after":  balance,
				"unix-timestamp": 1700000000,
			})
		},
	})

	numbers := make([]string, 250)
	for i := range numbers {
		numbers[i] = fmt.Sprintf("9655%07d", i)
	}

	withMockServer(t, handler, func(c *KwtSMS) {
		result, err := c.Send(strings.Join(numbers, ","), "Bulk test", "")
		if err != nil {
			t.Fatalf("Send error: %v", err)
		}
		if result.Result != "OK" {
			t.Fatalf("expected OK, got %s (%s)", result.Result, result.Code)
		}
		if len(batchSizes) != 2 || batchSizes[0] != 200 || batchSizes[1] != 50 {
			t.Errorf("expected batches [200 50], got %v", batchSizes)
		}
		if result.Numbers != 250 || result.PointsCharged != 250 {
			t.Errorf("expected 250 numbers and 250 points, got %d and %d", result.Numbers, result.PointsCharged)
		}
		if result.BalanceAfter != 750 {
			t.Errorf("expected balance-after 750, got %.2f", result.BalanceAfter)
		}
		if result.MsgID != "batch1" {
			t.Errorf("expected first batch msg-id, got %q", result.MsgID)
		}
	})
}
