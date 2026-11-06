// Package kwtsms provides a Go client for the kwtSMS SMS API (kwtsms.com).
//
// Zero external dependencies. Uses only the Go standard library.
//
// Quick start:
//
//	sms, err := kwtsms.FromEnv("")
//	ok, balance, err := sms.Verify()
//	result, err := sms.Send("965XXXXXXXX", "Your OTP for MYAPP is: 123456", "")
//	report := sms.Validate([]string{"965XXXXXXXX", "+96512345678"})
//	balance := sms.Balance()
//
// Server timezone: Asia/Kuwait (GMT+3). unix-timestamp values in API responses
// are GMT+3 server time, not UTC. Log timestamps written by this client are
// always UTC ISO-8601.
package kwtsms

// 2026-11-05 07:14 +03: placeholder 965XXXXXXXX instead of a sample mobile number (package doc and action text), int-testnumber.

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Version is the library version, sent in the User-Agent. The release workflow fails when it differs from the tag.
// 2026-11-05 14:17 +03: now sent as kwtsms-go/<Version> so kwtSMS can count accounts per integration (#939).
const Version = "0.5.0"

// KwtSMS is the kwtSMS API client. Safe for concurrent use.
type KwtSMS struct {
	username   string
	password   string
	senderID   string
	testMode   bool
	logFile    string
	httpClient *http.Client
	mu         sync.Mutex
	balance    *float64
	purchased  *float64
	userAgent  string // 2026-11-05 14:17 +03: User-Agent sent on every call (#939)
}

// Option configures a KwtSMS client.
type Option func(*KwtSMS)

// WithSenderID sets the default sender ID. Defaults to "KWT-SMS".
func WithSenderID(id string) Option {
	return func(c *KwtSMS) { c.senderID = id }
}

// WithTestMode enables test mode (messages queued but not delivered; credits held until you delete them from the Sending Queue).
// 2026-10-22 15:33 +03: corrected; test sends hold credits, they are not free.
func WithTestMode(enabled bool) Option {
	return func(c *KwtSMS) { c.testMode = enabled }
}

// WithLogFile sets the JSONL log file path. Set to "" to disable logging.
func WithLogFile(path string) Option {
	return func(c *KwtSMS) { c.logFile = path }
}

// WithHTTPClient sets a custom HTTP client. Useful for testing or proxies.
func WithHTTPClient(c *http.Client) Option {
	return func(k *KwtSMS) { k.httpClient = c }
}

// WithAppName sets the product token of the calling app or plugin, e.g. "kwtsms-wordpress/3.5.2",
// sent before the library token in the User-Agent. It is sanitized by BuildUserAgent.
// 2026-11-05 14:17 +03: added for per-integration counts (#939).
func WithAppName(name string) Option {
	return func(k *KwtSMS) { k.userAgent = BuildUserAgent(name) }
}

// New creates a new KwtSMS client with the given credentials.
//
// username and password are your kwtSMS API credentials (not your account
// mobile number). Options can override the default sender ID ("KWT-SMS"),
// enable test mode, or change the log file path ("kwtsms.log").
func New(username, password string, opts ...Option) (*KwtSMS, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	// Strip \r and \n that can sneak in from Windows .env files or shell sourcing
	cr := strings.NewReplacer("\r", "", "\n", "")
	c := &KwtSMS{
		username: cr.Replace(username),
		password: cr.Replace(password),
		senderID: "KWT-SMS",
		logFile:  "kwtsms.log",
		// 2026-11-05 14:17 +03: library token by default; WithAppName prepends the caller's token (#939).
		userAgent: BuildUserAgent(""),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// FromEnv creates a KwtSMS client from environment variables, falling back to
// a .env file. Pass "" for envFile to use the default ".env" path.
//
// Required env vars: KWTSMS_USERNAME, KWTSMS_PASSWORD
// Optional: KWTSMS_SENDER_ID (default "KWT-SMS"), KWTSMS_TEST_MODE ("1" to enable),
// KWTSMS_LOG_FILE (default "kwtsms.log")
func FromEnv(envFile string) (*KwtSMS, error) {
	if envFile == "" {
		envFile = ".env"
	}
	fileEnv := LoadEnvFile(envFile)

	get := func(key, fallback string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		if v, ok := fileEnv[key]; ok {
			return v
		}
		return fallback
	}

	username := get("KWTSMS_USERNAME", "")
	password := get("KWTSMS_PASSWORD", "")
	senderID := get("KWTSMS_SENDER_ID", "KWT-SMS")
	testMode := get("KWTSMS_TEST_MODE", "0") == "1"
	logFile := get("KWTSMS_LOG_FILE", "kwtsms.log")

	var missing []string
	if username == "" {
		missing = append(missing, "KWTSMS_USERNAME")
	}
	if password == "" {
		missing = append(missing, "KWTSMS_PASSWORD")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing credentials: %s", strings.Join(missing, ", "))
	}

	return New(username, password,
		WithSenderID(senderID),
		WithTestMode(testMode),
		WithLogFile(logFile),
	)
}

// CachedBalance returns the balance from the last Verify() or successful Send() call.
// Returns nil if no cached value exists.
func (c *KwtSMS) CachedBalance() *float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.balance == nil {
		return nil
	}
	v := *c.balance
	return &v
}

// CachedPurchased returns the total purchased credits from the last Verify() call.
// Returns nil if no cached value exists.
func (c *KwtSMS) CachedPurchased() *float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.purchased == nil {
		return nil
	}
	v := *c.purchased
	return &v
}

func (c *KwtSMS) creds() map[string]any {
	return map[string]any{"username": c.username, "password": c.password}
}

func (c *KwtSMS) setBalance(v float64) {
	c.mu.Lock()
	c.balance = &v
	c.mu.Unlock()
}

func (c *KwtSMS) setPurchased(v float64) {
	c.mu.Lock()
	c.purchased = &v
	c.mu.Unlock()
}

// Verify tests credentials by calling /balance/.
// Returns (ok, balance, error). On success, balance is the available balance.
// On failure, error describes the problem with an action to take.
// Never panics.
func (c *KwtSMS) Verify() (bool, float64, error) {
	data, err := c.request("balance", c.creds())
	if err != nil {
		return false, 0, err
	}

	result, _ := data["result"].(string)
	if result == "OK" {
		avail := toFloat64(data["available"])
		purch := toFloat64(data["purchased"])
		c.setBalance(avail)
		c.setPurchased(purch)
		return true, avail, nil
	}

	data = EnrichError(data)
	desc, _ := data["description"].(string)
	if desc == "" {
		desc, _ = data["code"].(string)
		if desc == "" {
			desc = "Unknown error"
		}
	}
	action, _ := data["action"].(string)
	if action != "" {
		desc = desc + " → " + action
	}
	return false, 0, fmt.Errorf("%s", desc)
}

// Balance returns the current balance via the /balance/ API call.
// Returns the cached value if the API call fails (and no cached value exists, returns 0 with error).
func (c *KwtSMS) Balance() (float64, error) {
	ok, bal, err := c.Verify()
	if ok {
		return bal, nil
	}
	c.mu.Lock()
	cached := c.balance
	c.mu.Unlock()
	if cached != nil {
		return *cached, nil
	}
	return 0, err
}

// Status checks the delivery status for a sent message via /status/.
//
// msgID is the message ID returned by Send() in result.MsgID.
func (c *KwtSMS) Status(msgID string) map[string]any {
	payload := c.creds()
	payload["msgid"] = msgID

	data, err := c.request("status", payload)
	if err != nil {
		return map[string]any{
			"result":      "ERROR",
			"code":        "NETWORK",
			"description": err.Error(),
			"action":      "Check your internet connection and try again.",
		}
	}
	return EnrichError(data)
}

// DLR retrieves delivery reports for a sent message via /dlr/.
// DLR only works for international (non-Kuwait) numbers.
//
// msgID is the message ID returned by Send() in result.MsgID.
func (c *KwtSMS) DLR(msgID string) map[string]any {
	payload := c.creds()
	payload["msgid"] = msgID

	data, err := c.request("dlr", payload)
	if err != nil {
		return map[string]any{
			"result":      "ERROR",
			"code":        "NETWORK",
			"description": err.Error(),
			"action":      "Check your internet connection and try again.",
		}
	}
	return EnrichError(data)
}

// SenderIDs lists sender IDs registered on this account via /senderid/.
// Never panics.
func (c *KwtSMS) SenderIDs() map[string]any {
	data, err := c.request("senderid", c.creds())
	if err != nil {
		return map[string]any{
			"result":      "ERROR",
			"code":        "NETWORK",
			"description": err.Error(),
			"action":      "Check your internet connection and try again.",
		}
	}

	result, _ := data["result"].(string)
	if result == "OK" {
		sids, _ := data["senderid"].([]any)
		strs := make([]string, 0, len(sids))
		for _, s := range sids {
			if str, ok := s.(string); ok {
				strs = append(strs, str)
			}
		}
		return map[string]any{
			"result":    "OK",
			"senderids": strs,
		}
	}
	return EnrichError(data)
}

// Coverage lists active country prefixes via /coverage/.
// Never panics.
func (c *KwtSMS) Coverage() map[string]any {
	data, err := c.request("coverage", c.creds())
	if err != nil {
		return map[string]any{
			"result":      "ERROR",
			"code":        "NETWORK",
			"description": err.Error(),
			"action":      "Check your internet connection and try again.",
		}
	}
	return EnrichError(data)
}

// Validate validates and normalizes phone numbers via /validate/.
//
// Numbers that fail local validation (empty, email, too short, too long, no digits)
// are rejected immediately before any API call is made.
func (c *KwtSMS) Validate(phones []string) ValidateResult {
	var validNormalized []string
	var preRejected []InvalidEntry

	for _, raw := range phones {
		v := ValidatePhoneInput(raw)
		if v.Valid {
			validNormalized = append(validNormalized, v.Normalized)
		} else {
			preRejected = append(preRejected, InvalidEntry{Input: raw, Error: v.Error})
		}
	}

	result := ValidateResult{
		OK:       []string{},
		ER:       make([]string, 0, len(preRejected)),
		NR:       []string{},
		Rejected: preRejected,
	}

	for _, r := range preRejected {
		result.ER = append(result.ER, r.Input)
	}

	if len(validNormalized) == 0 {
		if len(preRejected) == 1 {
			result.Error = preRejected[0].Error
		} else {
			result.Error = fmt.Sprintf("All %d phone numbers failed validation", len(preRejected))
		}
		return result
	}

	payload := c.creds()
	payload["mobile"] = strings.Join(validNormalized, ",")

	data, err := c.request("validate", payload)
	if err != nil {
		result.ER = append(validNormalized, result.ER...)
		result.Error = err.Error()
		return result
	}

	apiResult, _ := data["result"].(string)
	if apiResult == "OK" {
		mobile, _ := data["mobile"].(map[string]any)
		result.OK = toStringSlice(mobile["OK"])
		apiER := toStringSlice(mobile["ER"])
		result.ER = append(apiER, result.ER...)
		result.NR = toStringSlice(mobile["NR"])
		result.Raw = data
	} else {
		data = EnrichError(data)
		result.ER = append(validNormalized, result.ER...)
		result.Raw = data
		desc, _ := data["description"].(string)
		if desc == "" {
			desc, _ = data["code"].(string)
		}
		action, _ := data["action"].(string)
		if action != "" {
			desc = desc + " → " + action
		}
		result.Error = desc
	}

	return result
}

// Send sends an SMS to one or more numbers.
//
// mobile can be a single number or comma-separated list. Numbers are normalized
// automatically (strips +, 00, spaces, dashes, Arabic digits). Duplicates after
// normalization are deduplicated.
//
// message is cleaned automatically (strips emojis, hidden chars, HTML, converts
// Arabic digits to Latin).
//
// sender overrides the default sender ID for this call. Pass "" to use the default.
//
// For <= 200 numbers, returns a SendResult. For > 200 numbers, returns a BulkSendResult
// (accessed via the Bulk field).
func (c *KwtSMS) Send(mobile string, message string, sender string) (*SendResult, error) {
	return c.sendMulti(strings.Split(mobile, ","), message, sender)
}

// SendMulti sends an SMS to multiple numbers provided as a slice.
func (c *KwtSMS) SendMulti(mobiles []string, message string, sender string) (*SendResult, error) {
	return c.sendMulti(mobiles, message, sender)
}

func (c *KwtSMS) sendMulti(rawList []string, message string, sender string) (*SendResult, error) {
	effectiveSender := sender
	if effectiveSender == "" {
		effectiveSender = c.senderID
	}

	var validNumbers []string
	var invalid []InvalidEntry
	seen := make(map[string]bool)

	for _, raw := range rawList {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		v := ValidatePhoneInput(raw)
		if v.Valid {
			if !seen[v.Normalized] {
				seen[v.Normalized] = true
				validNumbers = append(validNumbers, v.Normalized)
			}
		} else {
			invalid = append(invalid, InvalidEntry{Input: raw, Error: v.Error})
		}
	}

	if len(validNumbers) == 0 {
		desc := "All phone numbers are invalid"
		if len(invalid) == 1 {
			desc = invalid[0].Error
		} else if len(invalid) > 1 {
			desc = fmt.Sprintf("All %d phone numbers are invalid", len(invalid))
		}
		r := &SendResult{
			Result:      "ERROR",
			Code:        "ERR_INVALID_INPUT",
			Description: desc,
			Invalid:     invalid,
		}
		if action, ok := APIErrors["ERR_INVALID_INPUT"]; ok {
			r.Action = action
		}
		return r, nil
	}

	cleaned := CleanMessage(message)
	if strings.TrimSpace(cleaned) == "" {
		r := &SendResult{
			Result:      "ERROR",
			Code:        "ERR009",
			Description: "Message is empty after cleaning (contained only emojis or invisible characters).",
		}
		if action, ok := APIErrors["ERR009"]; ok {
			r.Action = action
		}
		return r, nil
	}

	if len(validNumbers) > 200 {
		bulk := c.sendBulk(validNumbers, cleaned, effectiveSender)
		if len(invalid) > 0 {
			bulk.Invalid = invalid
		}
		// Return as SendResult with bulk info
		r := &SendResult{
			Result:        bulk.Result,
			Numbers:       bulk.Numbers,
			PointsCharged: bulk.PointsCharged,
			Invalid:       invalid,
		}
		if bulk.BalanceAfter != nil {
			r.BalanceAfter = *bulk.BalanceAfter
		}
		if len(bulk.MsgIDs) > 0 {
			r.MsgID = bulk.MsgIDs[0]
		}
		return r, nil
	}

	payload := c.creds()
	payload["sender"] = effectiveSender
	payload["mobile"] = strings.Join(validNumbers, ",")
	payload["message"] = cleaned
	if c.testMode {
		payload["test"] = "1"
	} else {
		payload["test"] = "0"
	}

	data, err := c.request("send", payload)
	if err != nil {
		r := &SendResult{
			Result:      "ERROR",
			Code:        "NETWORK",
			Description: err.Error(),
			Action:      "Check your internet connection and try again.",
			Invalid:     invalid,
		}
		return r, nil
	}

	result := mapToSendResult(data)
	if result.Result == "OK" {
		if ba, ok := data["balance-after"]; ok {
			c.setBalance(toFloat64(ba))
		}
	} else {
		data = EnrichError(data)
		result.Action, _ = data["action"].(string)
	}

	if len(invalid) > 0 {
		result.Invalid = invalid
	}

	return result, nil
}

// RetryWait is how long SendWithRetry waits before resending after ERR028.
// 2026-10-22 15:33 +03: added; replaces the hardcoded 16 s. ERR028 is opt-in per account
// (Duplicate SMS protection) and its window is set per account (10 seconds by default),
// so set this to your account's window if you changed it.
var RetryWait = 10 * time.Second

// SendWithRetry sends SMS and, only when maxRetries > 0, resends on ERR028 after waiting RetryWait.
// ERR028 means Duplicate SMS protection is on for the account and a number in the request was sent
// the same message within the account's window; the whole request was rejected and nothing was charged.
// All other errors are returned immediately.
func (c *KwtSMS) SendWithRetry(mobile string, message string, sender string, maxRetries int) (*SendResult, error) {
	// 2026-10-22 15:33 +03: maxRetries <= 0 now means no retry (was a silent default of 3),
	// so retrying happens only when the caller asks for it.
	if maxRetries < 0 {
		maxRetries = 0
	}

	result, err := c.Send(mobile, message, sender)
	if err != nil {
		return result, err
	}

	retries := 0
	for result.Code == "ERR028" && retries < maxRetries {
		time.Sleep(RetryWait) // 2026-10-22 15:33 +03: was a hardcoded 16 s; the window is per account
		result, err = c.Send(mobile, message, sender)
		if err != nil {
			return result, err
		}
		retries++
	}
	return result, nil
}

// sendBulk handles sending to >200 pre-normalized numbers in batches of 200.
func (c *KwtSMS) sendBulk(numbers []string, message string, sender string) BulkSendResult {
	const batchSize = 200
	const batchDelay = 500 * time.Millisecond
	err013Waits := []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second}

	var batches [][]string
	for i := 0; i < len(numbers); i += batchSize {
		end := i + batchSize
		if end > len(numbers) {
			end = len(numbers)
		}
		batches = append(batches, numbers[i:end])
	}

	var msgIDs []string
	var errors []BatchError
	totalNums := 0
	totalPts := 0
	var lastBalance *float64

	for i, batch := range batches {
		payload := c.creds()
		payload["sender"] = sender
		payload["mobile"] = strings.Join(batch, ",")
		payload["message"] = message
		if c.testMode {
			payload["test"] = "1"
		} else {
			payload["test"] = "0"
		}

		var data map[string]any
		for attempt := 0; attempt <= len(err013Waits); attempt++ {
			if attempt > 0 {
				time.Sleep(err013Waits[attempt-1])
			}

			var err error
			data, err = c.request("send", payload)
			if err != nil {
				errors = append(errors, BatchError{
					Batch:       i + 1,
					Code:        "NETWORK",
					Description: err.Error(),
				})
				data = nil
				break
			}

			code, _ := data["code"].(string)
			if code != "ERR013" || attempt == len(err013Waits) {
				break
			}
		}

		if data != nil {
			result, _ := data["result"].(string)
			switch result {
			case "OK":
				mid, _ := data["msg-id"].(string)
				msgIDs = append(msgIDs, mid)
				totalNums += int(toFloat64(data["numbers"]))
				totalPts += int(toFloat64(data["points-charged"]))
				if ba, ok := data["balance-after"]; ok {
					v := toFloat64(ba)
					lastBalance = &v
					c.setBalance(v)
				}
			case "ERROR":
				code, _ := data["code"].(string)
				desc, _ := data["description"].(string)
				errors = append(errors, BatchError{
					Batch:       i + 1,
					Code:        code,
					Description: desc,
				})
			}
		}

		if i < len(batches)-1 {
			time.Sleep(batchDelay)
		}
	}

	overall := "OK"
	if len(msgIDs) == 0 {
		overall = "ERROR"
	} else if len(msgIDs) < len(batches) {
		overall = "PARTIAL"
	}

	return BulkSendResult{
		Result:        overall,
		Bulk:          true,
		Batches:       len(batches),
		Numbers:       totalNums,
		PointsCharged: totalPts,
		BalanceAfter:  lastBalance,
		MsgIDs:        msgIDs,
		Errors:        errors,
	}
}

// toFloat64 safely converts an any value to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json_Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// json_Number is an alias to avoid importing encoding/json at top level
// while still handling json.Number from parsed responses.
type json_Number = interface{ Float64() (float64, error) }

// toStringSlice converts an any value (expected []any of strings) to []string.
func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// mapToSendResult converts a raw API response map to a SendResult.
func mapToSendResult(data map[string]any) *SendResult {
	r := &SendResult{
		Result: stringVal(data, "result"),
		Code:   stringVal(data, "code"),
	}
	r.Description, _ = data["description"].(string)
	r.MsgID, _ = data["msg-id"].(string)
	r.Numbers = int(toFloat64(data["numbers"]))
	r.PointsCharged = int(toFloat64(data["points-charged"]))
	r.BalanceAfter = toFloat64(data["balance-after"])
	r.UnixTimestamp = int64(toFloat64(data["unix-timestamp"]))
	return r
}

func stringVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
