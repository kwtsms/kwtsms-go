package kwtsms

// APIErrors maps every kwtSMS error code to a developer-friendly action message.
// Exported as read-only reference for callers who want to build custom error UIs.
var APIErrors = map[string]string{
	"ERR001":            "API is disabled on this account. Enable it at kwtsms.com → Account → API.",
	"ERR002":            "A required parameter is missing. Check that username, password, sender, mobile, and message are all provided.",
	"ERR003":            "Wrong API username or password. Check KWTSMS_USERNAME and KWTSMS_PASSWORD. These are your API credentials, not your account mobile number.",
	"ERR004":            "This account does not have API access. Contact kwtSMS support to enable it.",
	"ERR005":            "This account is blocked. Contact kwtSMS support.",
	// 2026-11-05 07:14 +03: placeholder 965XXXXXXXX instead of a sample mobile number, int-testnumber.
	"ERR006":            "No valid phone numbers. Make sure each number includes the country code (e.g., 965XXXXXXXX for Kuwait, not XXXXXXXX).",
	"ERR007":            "Too many numbers in a single request (maximum 200). Split into smaller batches.",
	"ERR008":            "This sender ID is banned or not found. Sender IDs are case sensitive (\"Kuwait\" is not the same as \"KUWAIT\"). Check your registered sender IDs at kwtsms.com.",
	"ERR009":            "Message is empty. Provide a non-empty message text.",
	"ERR010":            "Account balance is zero. Recharge credits at kwtsms.com.",
	"ERR011":            "Insufficient balance for this send. Buy more credits at kwtsms.com.",
	"ERR012":            "Message is too long (over 6 SMS pages). Shorten your message.",
	"ERR013":            "Send queue is full (1000 messages). Wait a moment and try again.",
	"ERR019":            "No delivery reports found for this message.",
	"ERR020":            "Message ID does not exist. Make sure you saved the msg-id from the send response.",
	"ERR021":            "No delivery report available for this message yet.",
	"ERR022":            "Delivery reports are not ready yet. Try again after 24 hours.",
	"ERR023":            "Unknown delivery report error. Contact kwtSMS support.",
	"ERR024":            "Your IP address is not in the API whitelist. Add it at kwtsms.com → Account → API → IP Lockdown, or disable IP lockdown.",
	"ERR025":            "Invalid phone number. Make sure the number includes the country code (e.g., 965XXXXXXXX for Kuwait, not XXXXXXXX).",
	"ERR026":            "This country is not activated on your account. Contact kwtSMS support to enable the destination country.",
	"ERR027":            "HTML tags are not allowed in the message. Remove any HTML content and try again.",
	// 2026-10-22 15:33 +03: ERR028 is opt-in per account with a per-account window, not a fixed 15 s.
	"ERR028":            "Sent to the same number too soon. Duplicate SMS protection is on for this account, so the whole request was rejected and nothing was charged. Wait for the account's window (10 seconds by default) before resending.",
	"ERR029":            "Message ID does not exist or is incorrect.",
	"ERR030":            "Message is stuck in the send queue with an error. Delete it at kwtsms.com → Queue to recover credits.",
	"ERR031":            "Message rejected: bad language detected.",
	"ERR032":            "Message rejected: spam detected.",
	"ERR033":            "No active coverage found. Contact kwtSMS support.",
	"ERR_INVALID_INPUT": "One or more phone numbers are invalid. See details above.",
}

// InvalidEntry represents a phone number that failed local pre-validation.
type InvalidEntry struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

// SendResult is the structured response from a single send operation (<= 200 numbers).
type SendResult struct {
	Result        string         `json:"result"`
	Code          string         `json:"code,omitempty"`
	Description   string         `json:"description,omitempty"`
	Action        string         `json:"action,omitempty"`
	MsgID         string         `json:"msg-id,omitempty"`
	Numbers       int            `json:"numbers,omitempty"`
	PointsCharged int            `json:"points-charged,omitempty"`
	BalanceAfter  float64        `json:"balance-after,omitempty"`
	UnixTimestamp int64          `json:"unix-timestamp,omitempty"`
	Invalid       []InvalidEntry `json:"invalid,omitempty"`
}

// BulkSendResult is the aggregated response from sending to >200 numbers.
type BulkSendResult struct {
	Result        string         `json:"result"`
	Bulk          bool           `json:"bulk"`
	Batches       int            `json:"batches"`
	Numbers       int            `json:"numbers"`
	PointsCharged int            `json:"points-charged"`
	BalanceAfter  *float64       `json:"balance-after"`
	MsgIDs        []string       `json:"msg-ids"`
	Errors        []BatchError   `json:"errors"`
	Invalid       []InvalidEntry `json:"invalid,omitempty"`
}

// BatchError records an error from a single batch within a bulk send.
type BatchError struct {
	Batch       int    `json:"batch"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ValidateResult is the structured response from validating phone numbers.
type ValidateResult struct {
	OK       []string         `json:"ok"`
	ER       []string         `json:"er"`
	NR       []string         `json:"nr"`
	Raw      map[string]any   `json:"raw"`
	Error    string           `json:"error,omitempty"`
	Rejected []InvalidEntry   `json:"rejected"`
}

// EnrichError adds an "action" field to an API error response map.
// Returns a new map. Has no effect on OK responses.
func EnrichError(data map[string]any) map[string]any {
	result, _ := data["result"].(string)
	if result != "ERROR" {
		return data
	}
	code, _ := data["code"].(string)
	action, ok := APIErrors[code]
	if !ok {
		return data
	}
	enriched := make(map[string]any, len(data)+1)
	for k, v := range data {
		enriched[k] = v
	}
	enriched["action"] = action
	return enriched
}
