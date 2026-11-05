package kwtsms

import (
	"strings"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"   ", ""},
		{"abc", ""},
		{"---", ""},

		// Domestic trunk prefix stripping
		{"9660559876543", "966559876543"},     // Saudi 9660... -> 966...
		{"+9660559876543", "966559876543"},    // Saudi with + prefix
		{"009660559876543", "966559876543"},   // Saudi with 00 prefix
		{"971050123456", "97150123456"},        // UAE trunk 0 stripped
		{"2001012345678", "201012345678"},      // Egypt trunk 0 stripped: 20+01012345678 -> 20+1012345678
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeSaudiTrunkPrefix(t *testing.T) {
	// The specific case called out: 9660559... -> 966559...
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Saudi with trunk 0", "9660559876543", "966559876543"},
		{"Saudi +country 0local", "+9660559876543", "966559876543"},
		{"Saudi 00country 0local", "009660559876543", "966559876543"},
		{"Saudi already clean", "966559876543", "966559876543"},
		{"Saudi multiple leading 0s", "96600559876543", "966559876543"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindCountryCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// 2026-11-05 07:14 +03: placeholder, prefix lookup only, int-testnumber
		{testNumberPlaceholder, "965"}, // Kuwait (3-digit)
		{"966559876543", "966"}, // Saudi (3-digit)
		{"44712345678", "44"},   // UK (2-digit)
		{"12125551234", "1"},    // USA (1-digit)
		{"9821234567890", "98"}, // Iran (2-digit)
		{"8613812345678", "86"}, // China (2-digit)
		{"", ""},
		{"999", ""},             // no matching country
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FindCountryCode(tt.input)
			if got != tt.want {
				t.Errorf("FindCountryCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePhoneFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantSub   string // substring in error, empty if valid
	}{
		// Valid numbers
		{"Saudi valid", "966559876543", true, ""},
		{"UAE valid", "971501234567", true, ""},
		{"USA valid", "12125551234", true, ""},
		{"UK valid", "447712345678", true, ""},
		{"Egypt valid", "201012345678", true, ""},  // 20 + 1012345678 (10 local digits, starts with 1)

		// Invalid length
		{"Kuwait too short", "9655123456", false, "expected 8 digits after +965"},
		{"Kuwait too long", "965512345678", false, "expected 8 digits after +965"},
		{"Saudi wrong length", "96655987654", false, "expected 9 digits after +966"},

		// Invalid mobile prefix
		{"Kuwait wrong prefix 1x", "96512345678", false, "must start with 4, 5, 6, 9"},
		{"Kuwait wrong prefix 2x", "96522345678", false, "must start with 4, 5, 6, 9"},
		{"Kuwait wrong prefix 3x", "96532345678", false, "must start with 4, 5, 6, 9"},
		{"Kuwait wrong prefix 7x", "96572345678", false, "must start with 4, 5, 6, 9"},
		{"Kuwait wrong prefix 8x", "96582345678", false, "must start with 4, 5, 6, 9"},
		{"Saudi wrong prefix", "966612345678", false, "must start with 5"},
		{"UAE wrong prefix", "971612345678", false, "must start with 5"},

		// Unknown country: passes through
		{"unknown country", "9991234567890", true, ""},

		// Countries with length-only rules (no prefix check)
		{"Belgium valid", "32471234567", true, ""},
		{"Poland valid", "48512345678", true, ""},
		{"Denmark valid", "4531234567", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneFormat(tt.input)
			if tt.wantValid && err != "" {
				t.Errorf("ValidatePhoneFormat(%q) = %q, want valid", tt.input, err)
			}
			if !tt.wantValid && err == "" {
				t.Errorf("ValidatePhoneFormat(%q) = valid, want error containing %q", tt.input, tt.wantSub)
			}
			if tt.wantSub != "" && !strings.Contains(err, tt.wantSub) {
				t.Errorf("ValidatePhoneFormat(%q) = %q, want substring %q", tt.input, err, tt.wantSub)
			}
		})
	}
}

func TestValidatePhoneInput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantValid  bool
		wantNorm   string
		wantErrSub string // substring expected in error message
	}{
		// Basic validation (same as before)
		{"valid minimum 7 digits", "9991234", true, "9991234", ""},
		{"valid maximum 15 digits", "999123456789012", true, "999123456789012", ""},
		{"empty", "", false, "", "required"},
		{"blank spaces", "   ", false, "", "required"},
		{"email address", "user@gmail.com", false, "", "email address"},
		{"no digits", "abc", false, "", "no digits found"},
		{"dashes only", "---", false, "", "no digits found"},
		{"too short 3 digits", "123", false, "123", "too short"},
		{"too short 6 digits", "123456", false, "123456", "too short"},
		{"too long 16 digits", "1234567890123456", false, "1234567890123456", "too long"},
		{"too short 1 digit", "5", false, "5", "1 digit"},
		{"email with numbers", "test123@gmail.com", false, "", "email address"},

		// Country-specific validation via ValidatePhoneInput
		{"Kuwait invalid prefix 1x", "+96512345678", false, "96512345678", "must start with"},
		{"Saudi valid", "+966559876543", true, "966559876543", ""},
		{"Saudi with trunk 0", "+9660559876543", true, "966559876543", ""},
		{"Saudi invalid prefix", "+966612345678", false, "966612345678", "must start with 5"},
		{"Kuwait wrong length", "+9655123456", false, "9655123456", "expected 8 digits"},
		{"UAE valid", "+971501234567", true, "971501234567", ""},
		{"Egypt valid", "+201012345678", true, "201012345678", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ValidatePhoneInput(tt.input)
			if v.Valid != tt.wantValid {
				t.Errorf("ValidatePhoneInput(%q).Valid = %v, want %v (error: %s)", tt.input, v.Valid, tt.wantValid, v.Error)
			}
			if tt.wantNorm != "" && v.Normalized != tt.wantNorm {
				t.Errorf("ValidatePhoneInput(%q).Normalized = %q, want %q", tt.input, v.Normalized, tt.wantNorm)
			}
			if tt.wantErrSub != "" && !strings.Contains(v.Error, tt.wantErrSub) {
				t.Errorf("ValidatePhoneInput(%q).Error = %q, want substring %q", tt.input, v.Error, tt.wantErrSub)
			}
			if tt.wantValid && v.Error != "" {
				t.Errorf("ValidatePhoneInput(%q).Error = %q, want empty for valid input", tt.input, v.Error)
			}
		})
	}
}

func TestValidatePhoneFormatGCC(t *testing.T) {
	// Thorough GCC country validation
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		// Kuwait (valid prefixes: TestKuwaitValidPrefixes)
		{"KW 0x invalid", "96501234567", false},

		// Saudi Arabia
		{"SA 5x valid", "966512345678", true},
		{"SA 3x invalid", "966312345678", false},

		// UAE
		{"AE 5x valid", "971512345678", true},
		{"AE 3x invalid", "971312345678", false},

		// Bahrain
		{"BH 3x valid", "97331234567", true},
		{"BH 6x valid", "97361234567", true},
		{"BH 5x invalid", "97351234567", false},

		// Qatar
		{"QA 3x valid", "97431234567", true},
		{"QA 5x valid", "97451234567", true},
		{"QA 6x valid", "97461234567", true},
		{"QA 7x valid", "97471234567", true},
		{"QA 1x invalid", "97411234567", false},

		// Oman
		{"OM 7x valid", "96871234567", true},
		{"OM 9x valid", "96891234567", true},
		{"OM 5x invalid", "96851234567", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneFormat(tt.input)
			if tt.wantValid && err != "" {
				t.Errorf("ValidatePhoneFormat(%q) = %q, want valid", tt.input, err)
			}
			if !tt.wantValid && err == "" {
				t.Errorf("ValidatePhoneFormat(%q) = valid, want error", tt.input)
			}
		})
	}
}

// 2026-11-05 07:14 +03: Kuwait cases moved here from the tables above; the
// number comes from KWTSMS_TEST_NUMBER (testNumber, skips when unset) and its
// format variants are built at runtime. Invalid-length samples changed to
// 9655123456 / 965512345678. Subtest names and failure messages never contain
// the number. int-testnumber.
func TestNormalizePhoneKuwaitVariants(t *testing.T) {
	n := testNumber(t)
	cc, a, b := splitKW(n)
	tests := []struct {
		name  string
		input string
	}{
		{"plus prefix", "+" + n},
		{"00 prefix", "00" + n},
		{"spaces", cc + " " + a + " " + b},
		{"dashes", cc + "-" + a + "-" + b},
		{"parens and dash", "(" + cc + ") " + a + "-" + b},
		{"Arabic-Indic digits", toArabicIndic(n)},
		{"Persian digits", toPersian(n)},
		{"dots", "+" + cc + "." + a + "." + b},
		{"triple leading zeros", "000" + n},
		{"already clean", n},
		{"padded with spaces", "  +" + cc + " " + a + " " + b + "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizePhone(tt.input); got != n {
				t.Errorf("NormalizePhone(%s) did not return the test number", tt.name)
			}
		})
	}
}

// 2026-11-05 07:14 +03: Kuwait valid cases for ValidatePhoneFormat and
// ValidatePhoneInput, from KWTSMS_TEST_NUMBER. int-testnumber.
func TestValidatePhoneKuwaitTestNumber(t *testing.T) {
	n := testNumber(t)
	if err := ValidatePhoneFormat(n); err != "" {
		t.Errorf("ValidatePhoneFormat(test number) = %q, want valid", err)
	}
	inputs := []struct {
		name  string
		input string
	}{
		{"plus prefix", "+" + n},
		{"00 prefix", "00" + n},
		{"Arabic-Indic digits", toArabicIndic(n)},
	}
	for _, tt := range inputs {
		t.Run(tt.name, func(t *testing.T) {
			v := ValidatePhoneInput(tt.input)
			if !v.Valid {
				t.Errorf("ValidatePhoneInput(%s).Valid = false (error: %s)", tt.name, v.Error)
			}
			if v.Normalized != n {
				t.Errorf("ValidatePhoneInput(%s).Normalized is not the test number", tt.name)
			}
		})
	}
}

// 2026-11-05 11:37 +03: the valid-prefix cases (4, 5, 6, 9) used sequential sample mobiles that
// may belong to real people. They are built at runtime from KWTSMS_TEST_NUMBER:
// each prefix plus the test number's last 7 digits, checked locally, never sent
// or printed. int-testnumber.
func TestKuwaitValidPrefixes(t *testing.T) {
	n := testNumber(t)
	for _, p := range []string{"4", "5", "6", "9"} {
		t.Run("prefix "+p, func(t *testing.T) {
			if err := ValidatePhoneFormat("965" + p + n[4:]); err != "" {
				t.Errorf("ValidatePhoneFormat(prefix %s) rejected a valid Kuwait mobile shape", p)
			}
		})
	}
}
