package kwtsms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://www.kwtsms.com/API/"

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// BuildUserAgent returns the User-Agent "[app token ]kwtsms-go/<Version>". The app token is caller input:
// anything outside printable ASCII is removed (CR/LF would split the header), whitespace runs become '-',
// leading/trailing '-' are trimmed, and it is cut to 64 characters. Empty means the library token only.
// 2026-11-05 14:17 +03: added so kwtSMS can count accounts per integration (#939).
func BuildUserAgent(appName string) string {
	var b strings.Builder
	for _, r := range appName {
		if r >= 0x20 && r <= 0x7e {
			b.WriteRune(r)
		}
	}
	app := strings.Trim(strings.Join(strings.Fields(b.String()), "-"), "-")
	if len(app) > 64 {
		app = app[:64]
	}
	lib := "kwtsms-go/" + Version
	if app == "" {
		return lib
	}
	return app + " " + lib
}

// UserAgent returns the User-Agent this client sends on every API call.
// 2026-11-05 14:17 +03: added for per-integration counts (#939).
func (c *KwtSMS) UserAgent() string {
	if c.userAgent == "" {
		return BuildUserAgent("")
	}
	return c.userAgent
}

// request POSTs JSON to a kwtSMS API endpoint and returns the parsed response.
// It reads 4xx/5xx response bodies (kwtSMS returns JSON error details in 403s).
// Logs every call to logFile if provided. Password is always masked in logs.
func (c *KwtSMS) request(endpoint string, payload map[string]any) (map[string]any, error) {
	url := baseURL + endpoint + "/"

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	entry := logEntry{
		Endpoint: endpoint,
		Request:  maskPassword(payload),
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		entry.Error = err.Error()
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// 2026-11-05 14:17 +03: kwtsms-go/<Version>, app token first when given (#939).
	req.Header.Set("User-Agent", c.UserAgent())

	hc := httpClient
	if c.httpClient != nil {
		hc = c.httpClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		entry.Error = fmt.Sprintf("Network error: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		entry.Error = fmt.Sprintf("Failed to read response: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		entry.Error = fmt.Sprintf("Invalid JSON response: %v", err)
		writeLog(c.logFile, entry)
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	entry.Response = data
	result, _ := data["result"].(string)
	entry.OK = result == "OK"
	writeLog(c.logFile, entry)

	return data, nil
}
