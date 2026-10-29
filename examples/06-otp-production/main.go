// Production-ready OTP HTTP server using kwtsms-go.
//
// Covers: rate limiting per phone, OTP expiry, resend cooldown,
// new code on resend, user-facing errors, balance tracking,
// transactional sender ID, and msg-id persistence.
//
// Run:
//
//	go run .
//
// Request OTP:
//
//	curl -X POST http://localhost:8080/otp/request -d '{"phone":"96598765432"}'
//
// Verify OTP:
//
//	curl -X POST http://localhost:8080/otp/verify -d '{"phone":"96598765432","code":"123456"}'
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

// OTP configuration
const (
	otpLength     = 6
	otpExpiry     = 4 * time.Minute  // OTP valid for 4 minutes
	resendCooldown = 4 * time.Minute // KNET standard: 4 minutes between resends
	maxPerHour    = 5                // Max OTP requests per phone per hour
	appName       = "MYAPP"          // REQUIRED: telecom compliance
	senderID      = "MY-SENDER"      // Use your Transactional sender ID, never KWT-SMS
)

// otpRecord stores a pending OTP for a phone number.
type otpRecord struct {
	Code      string
	ExpiresAt time.Time
	MsgID     string
}

// rateRecord tracks OTP request frequency per phone number.
type rateRecord struct {
	Count    int
	WindowStart time.Time
	LastSent time.Time
}

// In-memory stores. Use Redis or a database in production.
var (
	otpStore  = map[string]*otpRecord{}
	rateStore = map[string]*rateRecord{}
	mu        sync.Mutex
	sms       *kwtsms.KwtSMS
)

func main() {
	var err error
	sms, err = kwtsms.FromEnv("")
	if err != nil {
		log.Fatal(err)
	}

	// Verify credentials at startup
	ok, balance, verifyErr := sms.Verify()
	if !ok {
		log.Fatalf("credential verification failed: %v", verifyErr)
	}
	log.Printf("SMS client ready, balance: %.2f credits", balance)

	http.HandleFunc("/otp/request", handleOTPRequest)
	http.HandleFunc("/otp/verify", handleOTPVerify)

	log.Println("OTP server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Validate CAPTCHA token here before proceeding.
	// Without CAPTCHA, bots can drain your SMS balance.

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Validate phone number locally (no API call wasted)
	v := kwtsms.ValidatePhoneInput(req.Phone)
	if !v.Valid {
		jsonError(w, "Please enter a valid phone number in international format (e.g., +965 9876 5432).", http.StatusBadRequest)
		return
	}
	normalized := v.Normalized

	mu.Lock()
	defer mu.Unlock()

	// 2. Rate limiting per phone number
	rate, exists := rateStore[normalized]
	now := time.Now()

	if exists {
		// Reset window if older than 1 hour
		if now.Sub(rate.WindowStart) > time.Hour {
			rate.Count = 0
			rate.WindowStart = now
		}

		// Check hourly limit
		if rate.Count >= maxPerHour {
			jsonError(w, fmt.Sprintf("Too many requests to this number. Please try again in %d minutes.",
				int(time.Hour.Minutes()-now.Sub(rate.WindowStart).Minutes())), http.StatusTooManyRequests)
			return
		}

		// Check resend cooldown
		if !rate.LastSent.IsZero() && now.Sub(rate.LastSent) < resendCooldown {
			remaining := resendCooldown - now.Sub(rate.LastSent)
			jsonError(w, fmt.Sprintf("Please wait %d seconds before requesting another code.",
				int(remaining.Seconds())), http.StatusTooManyRequests)
			return
		}
	} else {
		rate = &rateRecord{WindowStart: now}
		rateStore[normalized] = rate
	}

	// 3. Generate a new OTP (invalidates any previous code for this number)
	code, err := generateOTP(otpLength)
	if err != nil {
		log.Printf("OTP generation error: %v", err)
		jsonError(w, "SMS service is temporarily unavailable. Please try again later.", http.StatusInternalServerError)
		return
	}

	// 4. Send via kwtSMS with app name (telecom compliance)
	message := fmt.Sprintf("Your OTP for %s is: %s", appName, code)
	result, sendErr := sms.Send(normalized, message, senderID)
	if sendErr != nil {
		log.Printf("SMS send error: %v", sendErr)
		jsonError(w, "Could not connect to SMS service. Please check your internet connection and try again.", http.StatusBadGateway)
		return
	}

	if result.Result != "OK" {
		log.Printf("SMS API error: %s %s (action: %s)", result.Code, result.Description, result.Action)
		jsonError(w, userFacingMessage(result.Code), userFacingStatus(result.Code))
		return
	}

	// 5. Store OTP with expiry, save msg-id and balance
	otpStore[normalized] = &otpRecord{
		Code:      code,
		ExpiresAt: now.Add(otpExpiry),
		MsgID:     result.MsgID,
	}

	// 6. Update rate limiter
	rate.Count++
	rate.LastSent = now

	// 7. Save balance (in production, persist to database)
	log.Printf("OTP sent to %s, msg-id: %s, balance: %.2f", normalized, result.MsgID, result.BalanceAfter)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":         true,
		"message":         "OTP sent successfully",
		"expires_in_secs": int(otpExpiry.Seconds()),
		"resend_in_secs":  int(resendCooldown.Seconds()),
	})
}

func handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	v := kwtsms.ValidatePhoneInput(req.Phone)
	if !v.Valid {
		jsonError(w, "Invalid phone number", http.StatusBadRequest)
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		jsonError(w, "OTP code is required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	record, exists := otpStore[v.Normalized]
	if !exists {
		jsonError(w, "No OTP was requested for this number. Please request a new code.", http.StatusBadRequest)
		return
	}

	// Check expiry
	if time.Now().After(record.ExpiresAt) {
		delete(otpStore, v.Normalized)
		jsonError(w, "Code expired. Please request a new one.", http.StatusBadRequest)
		return
	}

	// Check code
	if record.Code != code {
		jsonError(w, "Invalid code. Please check and try again.", http.StatusBadRequest)
		return
	}

	// OTP verified, delete it (one-time use)
	delete(otpStore, v.Normalized)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "OTP verified successfully",
	})
}

// generateOTP returns a cryptographically random numeric code.
func generateOTP(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}

// userFacingMessage maps API error codes to messages safe for end users.
// System-level errors get generic messages. User-recoverable errors get helpful ones.
func userFacingMessage(code string) string {
	switch code {
	case "ERR006", "ERR025":
		return "Please enter a valid phone number in international format (e.g., +965 9876 5432)."
	case "ERR026":
		return "SMS delivery to this country is not available. Please contact support."
	case "ERR028":
		return "Please wait a moment before requesting another code."
	case "ERR031", "ERR032":
		return "Your message could not be sent. Please try again with different content."
	case "ERR013":
		return "SMS service is busy. Please try again in a few minutes."
	default:
		// ERR003, ERR010, ERR011, etc. are system errors.
		// Never tell the user about auth or balance problems.
		return "SMS service is temporarily unavailable. Please try again later."
	}
}

// userFacingStatus maps API error codes to HTTP status codes.
func userFacingStatus(code string) int {
	switch code {
	case "ERR006", "ERR025":
		return http.StatusBadRequest
	case "ERR028":
		return http.StatusTooManyRequests
	case "ERR013":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
	})
}
