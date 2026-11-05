// FILE:      testnumber_test.go
// CREATED:   2026-11-05 07:14 +03
// TASK:      int-testnumber (Boss 2026-11-05 00:33)
// AUTHOR:    M Alhassoun
// WHAT:      Shared test helper: the test phone number and its format variants.
// WHY:       Tests must not hardcode a phone number. The number comes from the
//            KWTSMS_TEST_NUMBER environment variable (an international Kuwait
//            mobile, digits only, 965XXXXXXXX) and every test that needs one
//            is skipped when it is unset or malformed.
// USAGE:     KWTSMS_TEST_NUMBER=965XXXXXXXX go test ./...
//            CI reads it from the repository secret of the same name.
// DEPENDS:   stdlib only (os, strings, testing). No build tag, so the
//            integration tests (-tags integration) use it too.

package kwtsms

import (
	"os"
	"strings"
	"testing"
)

// testNumberPlaceholder is for values that are only passed through unchanged
// (for example a number echoed back in a mocked API response).
// 2026-11-05 07:14 +03: placeholder for pass-through fields, int-testnumber.
const testNumberPlaceholder = "965XXXXXXXX"

// testNumber returns KWTSMS_TEST_NUMBER (965 plus 8 digits) or skips the test.
// Never log or print the returned value.
// 2026-11-05 07:14 +03: single source of the test number, int-testnumber.
func testNumber(t *testing.T) string {
	t.Helper()
	n := strings.TrimSpace(os.Getenv("KWTSMS_TEST_NUMBER"))
	if len(n) != 11 || !strings.HasPrefix(n, "965") || !isASCIIDigits(n) {
		t.Skip("KWTSMS_TEST_NUMBER not set (digits only, 965 plus 8 digits)")
	}
	return n
}

// isASCIIDigits reports whether s is non-empty and only 0-9.
// 2026-11-05 07:14 +03: validation for testNumber, int-testnumber.
func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// shiftDigits maps Latin digits to the digit block starting at zero.
// 2026-11-05 07:14 +03: format variants built at runtime, int-testnumber.
func shiftDigits(s string, zero rune) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return zero + (r - '0')
		}
		return r
	}, s)
}

// toArabicIndic converts Latin digits to Arabic-Indic (U+0660-0669).
// 2026-11-05 07:14 +03: int-testnumber.
func toArabicIndic(s string) string { return shiftDigits(s, '٠') }

// toPersian converts Latin digits to Extended Arabic-Indic / Persian (U+06F0-06F9).
// 2026-11-05 07:14 +03: int-testnumber.
func toPersian(s string) string { return shiftDigits(s, '۰') }

// splitKW splits a 965XXXXXXXX number into country code, first 4 and last 4 local digits.
// 2026-11-05 07:14 +03: builds spaced and dashed variants, int-testnumber.
func splitKW(n string) (cc, a, b string) {
	return n[:3], n[3:7], n[7:]
}
