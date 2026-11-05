// FILE:      useragent_test.go
// CREATED:   2026-11-05 14:18 +03
// TASK:      Vikunja #939 (int-measure-signups)
// AUTHOR:    M Alhassoun
// WHAT:      Unit tests for the User-Agent kwtsms-go/<Version> and the WithAppName option.
// WHY:       kwtSMS counts accounts per integration from the User-Agent; it must be exact and header-safe.
// USAGE:     go test -run UserAgent ./...
// DEPENDS:   net/http/httptest and rewriteTransport (mock_test.go); no network.

package kwtsms

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestUserAgentDefault(t *testing.T) {
	c, err := New("go_api_user", "go_api_pass", WithLogFile(""))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.UserAgent(), "kwtsms-go/"+Version; got != want {
		t.Fatalf("UserAgent() = %q, want %q", got, want)
	}
	if !regexp.MustCompile(`^kwtsms-go/[0-9][0-9A-Za-z.+-]*$`).MatchString(c.UserAgent()) {
		t.Fatalf("UserAgent() %q does not match the token format", c.UserAgent())
	}
}

func TestUserAgentAppNamePrepended(t *testing.T) {
	c, err := New("go_api_user", "go_api_pass", WithLogFile(""), WithAppName("kwtsms-wordpress/3.5.2"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.UserAgent(), "kwtsms-wordpress/3.5.2 kwtsms-go/"+Version; got != want {
		t.Fatalf("UserAgent() = %q, want %q", got, want)
	}
}

func TestUserAgentSanitized(t *testing.T) {
	lib := "kwtsms-go/" + Version
	ua := BuildUserAgent("my app\r\nX-Injected: 1")
	if strings.ContainsAny(ua, "\r\n") || strings.Contains(ua, " X-Injected") || !strings.HasSuffix(ua, " "+lib) {
		t.Fatalf("BuildUserAgent did not sanitize: %q", ua)
	}
	if got := BuildUserAgent("  - -  "); got != lib {
		t.Fatalf("BuildUserAgent(blank) = %q, want %q", got, lib)
	}
	if n := len(strings.Split(BuildUserAgent(strings.Repeat("a", 200)), " ")[0]); n != 64 {
		t.Fatalf("app token length = %d, want 64", n)
	}
}

func TestUserAgentHeaderSent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		jsonResponse(w, map[string]any{"result": "OK", "available": 1.0, "purchased": 1.0})
	}))
	defer server.Close()
	mockHTTP := &http.Client{Transport: &rewriteTransport{base: http.DefaultTransport, fromURL: baseURL, toURL: server.URL + "/API/"}}

	c, err := New("go_api_user", "go_api_pass", WithLogFile(""), WithHTTPClient(mockHTTP), WithAppName("my-shop/2.1.0"))
	if err != nil {
		t.Fatal(err)
	}
	if ok, _, err := c.Verify(); !ok || err != nil {
		t.Fatalf("Verify() = %v, %v", ok, err)
	}
	if want := "my-shop/2.1.0 kwtsms-go/" + Version; got != want {
		t.Fatalf("User-Agent header = %q, want %q", got, want)
	}
}
