// Example 04: Minimal HTTP handler for sending SMS.
// Accepts POST JSON {"phone":"...","message":"..."}, validates the input,
// sends the SMS, and returns a JSON response. Demonstrates proper error
// handling with user-facing messages instead of raw API errors.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	kwtsms "github.com/kwtsms/kwtsms-go"
)

// smsRequest is the expected JSON body for POST /send.
type smsRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// smsResponse is the JSON response returned to the client.
type smsResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	MsgID   string `json:"msg_id,omitempty"`
}

// smsClient is the shared kwtsms client, initialized once at startup.
var smsClient *kwtsms.KwtSMS

func main() {
	var err error
	smsClient, err = kwtsms.FromEnv("")
	if err != nil {
		fmt.Println("Failed to load credentials:", err)
		os.Exit(1)
	}

	// Verify credentials at startup, not per-request.
	ok, balance, err := smsClient.Verify()
	if err != nil || !ok {
		fmt.Println("Account verification failed:", err)
		os.Exit(1)
	}
	fmt.Printf("Account verified. Balance: %.2f\n", balance)

	http.HandleFunc("/send", handleSend)

	addr := ":8080"
	fmt.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST.
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, smsResponse{
			OK:      false,
			Message: "Only POST is allowed.",
		})
		return
	}

	// TODO: Add rate limiting here. For production, use a middleware like
	// golang.org/x/time/rate or a per-IP token bucket to prevent abuse.
	// Example: allow 10 requests per minute per IP.

	// Parse the request body.
	var req smsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, smsResponse{
			OK:      false,
			Message: "Invalid JSON body.",
		})
		return
	}

	// Validate the phone number before calling the API.
	v := kwtsms.ValidatePhoneInput(req.Phone)
	if !v.Valid {
		writeJSON(w, http.StatusBadRequest, smsResponse{
			OK:      false,
			Message: v.Error,
		})
		return
	}

	// Clean the message and check for empty result.
	cleaned := kwtsms.CleanMessage(req.Message)
	if strings.TrimSpace(cleaned) == "" {
		writeJSON(w, http.StatusBadRequest, smsResponse{
			OK:      false,
			Message: "Message is empty after removing unsupported characters.",
		})
		return
	}

	// Send the SMS.
	result, err := smsClient.Send(v.Normalized, req.Message, "")
	if err != nil {
		// Network or internal error. Do not expose raw errors to the user.
		writeJSON(w, http.StatusServiceUnavailable, smsResponse{
			OK:      false,
			Message: "SMS service is temporarily unavailable. Please try again.",
		})
		return
	}

	if result.Result == "OK" {
		writeJSON(w, http.StatusOK, smsResponse{
			OK:      true,
			Message: "SMS sent successfully.",
			MsgID:   result.MsgID,
		})
		return
	}

	// Map API error codes to user-facing messages.
	// Never expose raw API error codes or internal details to end users.
	userMessage := mapErrorToUserMessage(result.Code)
	writeJSON(w, http.StatusBadRequest, smsResponse{
		OK:      false,
		Message: userMessage,
	})
}

// mapErrorToUserMessage converts an API error code to a safe, user-facing message.
func mapErrorToUserMessage(code string) string {
	switch code {
	case "ERR006", "ERR025":
		return "The phone number is not valid. Please check and try again."
	case "ERR009":
		return "The message cannot be empty."
	case "ERR010", "ERR011":
		return "SMS service is temporarily unavailable. Please try again later."
	case "ERR012":
		return "The message is too long. Please shorten it and try again."
	case "ERR027":
		return "HTML content is not allowed in SMS messages."
	case "ERR028":
		return "Please wait a moment before sending another message to the same number."
	case "ERR031", "ERR032":
		return "The message was rejected. Please revise the content."
	default:
		return "Failed to send SMS. Please try again later."
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
