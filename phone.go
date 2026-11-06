package kwtsms

import (
	"fmt"
	"strings"
	"unicode"
)

// arabicToLatin maps Arabic-Indic (U+0660-U+0669) and Extended Arabic-Indic /
// Persian (U+06F0-U+06F9) digits to their Latin equivalents.
var arabicToLatin = map[rune]rune{
	'٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4',
	'٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
	'۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4',
	'۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
}

// PhoneRule defines country-specific phone number validation rules.
// LocalLengths lists valid digit counts AFTER the country code.
// MobileStartDigits lists valid first character(s) of the local number.
type PhoneRule struct {
	LocalLengths      []int
	MobileStartDigits []string
}

// PhoneRules maps country codes to their phone number validation rules.
// Countries not listed here pass through with generic E.164 validation (7-15 digits).
//
// Sources (verified across 3+ per country):
//   - ITU-T E.164 / National Numbering Plans (itu.int)
//   - Wikipedia "Telephone numbers in [Country]" articles
//   - HowToCallAbroad.com country dialing guides
var PhoneRules = map[string]PhoneRule{
	// === GCC ===
	"965": {LocalLengths: []int{8}, MobileStartDigits: []string{"4", "5", "6", "9"}}, // Kuwait: 4x=Virgin/STC, 5x=STC/Zain, 6x=Ooredoo, 9x=Zain
	"966": {LocalLengths: []int{9}, MobileStartDigits: []string{"5"}},                // Saudi Arabia: 50-59
	"971": {LocalLengths: []int{9}, MobileStartDigits: []string{"5"}},                // UAE: 50,52-56,58
	"973": {LocalLengths: []int{8}, MobileStartDigits: []string{"3", "6"}},           // Bahrain: 3x,6x
	"974": {LocalLengths: []int{8}, MobileStartDigits: []string{"3", "5", "6", "7"}}, // Qatar: 33,55,66,77
	"968": {LocalLengths: []int{8}, MobileStartDigits: []string{"7", "9"}},           // Oman: 7x,9x
	// === Levant ===
	"962": {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},              // Jordan: 75,77,78,79
	"961": {LocalLengths: []int{7, 8}, MobileStartDigits: []string{"3", "7", "8"}}, // Lebanon: 3x (legacy 7-digit), 7x/81 (8-digit)
	"970": {LocalLengths: []int{9}, MobileStartDigits: []string{"5"}},              // Palestine: 56=Jawwal, 59=Ooredoo
	"964": {LocalLengths: []int{10}, MobileStartDigits: []string{"7"}},             // Iraq: 75-79
	"963": {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},              // Syria: 93-96,98,99
	// === Other Arab ===
	"967": {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},                // Yemen: 70,71,73,77
	"20":  {LocalLengths: []int{10}, MobileStartDigits: []string{"1"}},               // Egypt: 10,11,12,15
	"218": {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},                // Libya: 91-95
	"216": {LocalLengths: []int{8}, MobileStartDigits: []string{"2", "4", "5", "9"}}, // Tunisia: 2x,4x=MVNO,5x,9x
	"212": {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}},           // Morocco: 6x,7x
	"213": {LocalLengths: []int{9}, MobileStartDigits: []string{"5", "6", "7"}},      // Algeria: 5x,6x,7x
	"249": {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},                // Sudan: 90,91,92,96,99
	// === Non-Arab Middle East ===
	"98":  {LocalLengths: []int{10}, MobileStartDigits: []string{"9"}}, // Iran: 9x
	"90":  {LocalLengths: []int{10}, MobileStartDigits: []string{"5"}}, // Turkey: 5x
	"972": {LocalLengths: []int{9}, MobileStartDigits: []string{"5"}},  // Israel: 50,52-55,58
	// === South Asia ===
	"91":  {LocalLengths: []int{10}, MobileStartDigits: []string{"6", "7", "8", "9"}}, // India: 6-9x
	"92":  {LocalLengths: []int{10}, MobileStartDigits: []string{"3"}},                // Pakistan: 3x
	"880": {LocalLengths: []int{10}, MobileStartDigits: []string{"1"}},                // Bangladesh: 1x
	"94":  {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},                 // Sri Lanka: 70-78
	"960": {LocalLengths: []int{7}, MobileStartDigits: []string{"7", "9"}},            // Maldives: 7x,9x
	// === East Asia ===
	"86":  {LocalLengths: []int{11}, MobileStartDigits: []string{"1"}},           // China: 13-19x
	"81":  {LocalLengths: []int{10}, MobileStartDigits: []string{"7", "8", "9"}}, // Japan: 70,80,90
	"82":  {LocalLengths: []int{10}, MobileStartDigits: []string{"1"}},           // South Korea: 010
	"886": {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},            // Taiwan: 9x
	// === Southeast Asia ===
	"65":  {LocalLengths: []int{8}, MobileStartDigits: []string{"8", "9"}},                   // Singapore: 8x,9x
	"60":  {LocalLengths: []int{9, 10}, MobileStartDigits: []string{"1"}},                    // Malaysia: 1x (9 or 10 digits)
	"62":  {LocalLengths: []int{9, 10, 11, 12}, MobileStartDigits: []string{"8"}},            // Indonesia: 8x (variable length)
	"63":  {LocalLengths: []int{10}, MobileStartDigits: []string{"9"}},                       // Philippines: 9x
	"66":  {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "8", "9"}},              // Thailand: 6x,8x,9x
	"84":  {LocalLengths: []int{9}, MobileStartDigits: []string{"3", "5", "7", "8", "9"}},    // Vietnam: 3x,5x,7x,8x,9x
	"95":  {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},                        // Myanmar: 9x
	"855": {LocalLengths: []int{8, 9}, MobileStartDigits: []string{"1", "6", "7", "8", "9"}}, // Cambodia: mixed lengths
	"976": {LocalLengths: []int{8}, MobileStartDigits: []string{"6", "8", "9"}},              // Mongolia: 6x,8x,9x
	// === Europe ===
	"44":  {LocalLengths: []int{10}, MobileStartDigits: []string{"7"}},     // UK: 7x
	"33":  {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}}, // France: 6x,7x
	"49":  {LocalLengths: []int{10, 11}, MobileStartDigits: []string{"1"}}, // Germany: 15x,16x,17x
	"39":  {LocalLengths: []int{10}, MobileStartDigits: []string{"3"}},     // Italy: 3x
	"34":  {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}}, // Spain: 6x,7x
	"31":  {LocalLengths: []int{9}, MobileStartDigits: []string{"6"}},      // Netherlands: 6x
	"32":  {LocalLengths: []int{9}},                                        // Belgium: length only
	"41":  {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},      // Switzerland: 74-79
	"43":  {LocalLengths: []int{10}, MobileStartDigits: []string{"6"}},     // Austria: 65x-69x
	"47":  {LocalLengths: []int{8}, MobileStartDigits: []string{"4", "9"}}, // Norway: 4x,9x
	"48":  {LocalLengths: []int{9}},                                        // Poland: length only
	"30":  {LocalLengths: []int{10}, MobileStartDigits: []string{"6"}},     // Greece: 69x
	"420": {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}}, // Czech Republic: 6x,7x
	"46":  {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},      // Sweden: 7x
	"45":  {LocalLengths: []int{8}},                                        // Denmark: length only
	"40":  {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},      // Romania: 7x
	"36":  {LocalLengths: []int{9}},                                        // Hungary: length only
	"380": {LocalLengths: []int{9}},                                        // Ukraine: length only
	// === Americas ===
	"1":   {LocalLengths: []int{10}},                                       // USA/Canada: no mobile-specific prefix
	"52":  {LocalLengths: []int{10}},                                       // Mexico: no mobile-specific prefix since 2019
	"55":  {LocalLengths: []int{11}},                                       // Brazil: area code + 9 + subscriber
	"57":  {LocalLengths: []int{10}, MobileStartDigits: []string{"3"}},     // Colombia: 3x
	"54":  {LocalLengths: []int{10}, MobileStartDigits: []string{"9"}},     // Argentina: 9 + area + subscriber
	"56":  {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},      // Chile: 9x
	"58":  {LocalLengths: []int{10}, MobileStartDigits: []string{"4"}},     // Venezuela: 4x
	"51":  {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},      // Peru: 9x
	"593": {LocalLengths: []int{9}, MobileStartDigits: []string{"9"}},      // Ecuador: 9x
	"53":  {LocalLengths: []int{8}, MobileStartDigits: []string{"5", "6"}}, // Cuba: 5x,6x
	// === Africa ===
	"27":  {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7", "8"}},  // South Africa: 6x,7x,8x
	"234": {LocalLengths: []int{10}, MobileStartDigits: []string{"7", "8", "9"}}, // Nigeria: 70,71,80,81,90,91
	"254": {LocalLengths: []int{9}, MobileStartDigits: []string{"1", "7"}},       // Kenya: 1x,7x
	"233": {LocalLengths: []int{9}, MobileStartDigits: []string{"2", "5"}},       // Ghana: 2x,5x
	"251": {LocalLengths: []int{9}, MobileStartDigits: []string{"7", "9"}},       // Ethiopia: 7x,9x
	"255": {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}},       // Tanzania: 6x,7x
	"256": {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},            // Uganda: 7x
	"237": {LocalLengths: []int{9}, MobileStartDigits: []string{"6"}},            // Cameroon: 6x
	"225": {LocalLengths: []int{10}},                                             // Ivory Coast: length only
	"221": {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},            // Senegal: 7x
	"252": {LocalLengths: []int{9}, MobileStartDigits: []string{"6", "7"}},       // Somalia: 6x,7x
	"250": {LocalLengths: []int{9}, MobileStartDigits: []string{"7"}},            // Rwanda: 7x
	// === Oceania ===
	"61": {LocalLengths: []int{9}, MobileStartDigits: []string{"4"}},        // Australia: 4x
	"64": {LocalLengths: []int{8, 9, 10}, MobileStartDigits: []string{"2"}}, // New Zealand: 21,22,27-29
}

// CountryNames maps country codes to human-readable country names.
var CountryNames = map[string]string{
	// Middle East & North Africa
	"965": "Kuwait", "966": "Saudi Arabia", "971": "UAE", "973": "Bahrain",
	"974": "Qatar", "968": "Oman", "962": "Jordan", "961": "Lebanon",
	"970": "Palestine", "964": "Iraq", "963": "Syria", "967": "Yemen",
	"98": "Iran", "90": "Turkey", "972": "Israel", "20": "Egypt",
	"218": "Libya", "216": "Tunisia", "212": "Morocco", "213": "Algeria", "249": "Sudan",
	// Africa
	"27": "South Africa", "234": "Nigeria", "254": "Kenya", "233": "Ghana",
	"251": "Ethiopia", "255": "Tanzania", "256": "Uganda", "237": "Cameroon",
	"225": "Ivory Coast", "221": "Senegal", "252": "Somalia", "250": "Rwanda",
	// Europe
	"44": "UK", "33": "France", "49": "Germany", "39": "Italy", "34": "Spain",
	"31": "Netherlands", "32": "Belgium", "41": "Switzerland", "43": "Austria",
	"46": "Sweden", "47": "Norway", "45": "Denmark", "48": "Poland",
	"420": "Czech Republic", "30": "Greece", "40": "Romania", "36": "Hungary", "380": "Ukraine",
	// Americas
	"1": "USA/Canada", "52": "Mexico", "55": "Brazil", "57": "Colombia",
	"54": "Argentina", "56": "Chile", "58": "Venezuela", "51": "Peru",
	"593": "Ecuador", "53": "Cuba",
	// Asia
	"91": "India", "92": "Pakistan", "86": "China", "81": "Japan", "82": "South Korea",
	"886": "Taiwan", "65": "Singapore", "60": "Malaysia", "62": "Indonesia",
	"63": "Philippines", "66": "Thailand", "84": "Vietnam", "95": "Myanmar",
	"855": "Cambodia", "880": "Bangladesh", "94": "Sri Lanka", "960": "Maldives", "976": "Mongolia",
	// Oceania
	"61": "Australia", "64": "New Zealand",
}

// NormalizePhone converts a raw phone number to kwtSMS-accepted format:
// digits only, no leading zeros, domestic trunk prefix stripped.
//
// Steps: convert Arabic/Persian digits to Latin, strip all non-digit
// characters, strip leading zeros, strip domestic trunk prefix (e.g.
// 9660559... becomes 966559...).
func NormalizePhone(phone string) string {
	var b strings.Builder
	b.Grow(len(phone))
	for _, r := range phone {
		if latin, ok := arabicToLatin[r]; ok {
			b.WriteRune(latin)
		} else if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	normalized := strings.TrimLeft(b.String(), "0")

	// Strip domestic trunk prefix (leading 0 after country code).
	// e.g. 9660559... -> 966559..., 97105x -> 9715x, 20010x -> 2010x
	cc := FindCountryCode(normalized)
	if cc != "" {
		local := normalized[len(cc):]
		if strings.HasPrefix(local, "0") {
			normalized = cc + strings.TrimLeft(local, "0")
		}
	}

	return normalized
}

// PhoneValidation holds the result of ValidatePhoneInput.
type PhoneValidation struct {
	Valid      bool
	Error      string
	Normalized string
}

// ValidatePhoneInput validates a raw phone number input before sending to the API.
//
// Catches every common mistake without panicking:
//   - Empty or blank input
//   - Email address entered instead of a phone number
//   - Non-numeric text with no digits
//   - Too short after normalization (< 7 digits)
//   - Too long after normalization (> 15 digits, E.164 maximum)
//   - Country-specific length and mobile prefix validation
//   - Domestic trunk prefix stripping (e.g. 9660559... -> 966559...)
func ValidatePhoneInput(phone string) PhoneValidation {
	raw := strings.TrimSpace(fmt.Sprintf("%v", phone))

	if raw == "" {
		return PhoneValidation{Valid: false, Error: "Phone number is required", Normalized: ""}
	}

	if strings.Contains(raw, "@") {
		return PhoneValidation{
			Valid: false,
			Error: fmt.Sprintf("'%s' is an email address, not a phone number", raw),
		}
	}

	normalized := NormalizePhone(raw)

	if normalized == "" {
		return PhoneValidation{
			Valid: false,
			Error: fmt.Sprintf("'%s' is not a valid phone number, no digits found", raw),
		}
	}

	n := len(normalized)
	if n < 7 {
		digit := "digits"
		if n == 1 {
			digit = "digit"
		}
		return PhoneValidation{
			Valid:      false,
			Error:      fmt.Sprintf("'%s' is too short to be a valid phone number (%d %s, minimum is 7)", raw, n, digit),
			Normalized: normalized,
		}
	}

	if n > 15 {
		return PhoneValidation{
			Valid:      false,
			Error:      fmt.Sprintf("'%s' is too long to be a valid phone number (%d digits, maximum is 15)", raw, n),
			Normalized: normalized,
		}
	}

	// Country-specific format validation (length + mobile prefix)
	if err := ValidatePhoneFormat(normalized); err != "" {
		return PhoneValidation{Valid: false, Error: err, Normalized: normalized}
	}

	return PhoneValidation{Valid: true, Normalized: normalized}
}

// FindCountryCode extracts the country code prefix from a normalized phone number.
// Tries 3-digit codes first, then 2-digit, then 1-digit (longest match wins).
// Returns empty string if no known country code is found.
func FindCountryCode(normalized string) string {
	if len(normalized) >= 3 {
		if _, ok := PhoneRules[normalized[:3]]; ok {
			return normalized[:3]
		}
	}
	if len(normalized) >= 2 {
		if _, ok := PhoneRules[normalized[:2]]; ok {
			return normalized[:2]
		}
	}
	if len(normalized) >= 1 {
		if _, ok := PhoneRules[normalized[:1]]; ok {
			return normalized[:1]
		}
	}
	return ""
}

// ValidatePhoneFormat validates a normalized phone number against country-specific
// format rules. Checks local number length and mobile starting digits.
// Numbers with no matching country rules pass through (generic E.164 only).
// Returns empty string if valid, or an error message if invalid.
func ValidatePhoneFormat(normalized string) string {
	cc := FindCountryCode(normalized)
	if cc == "" {
		return ""
	}

	rule := PhoneRules[cc]
	local := normalized[len(cc):]
	country := CountryNames[cc]
	if country == "" {
		country = "+" + cc
	}

	// Check local number length
	validLen := false
	for _, l := range rule.LocalLengths {
		if len(local) == l {
			validLen = true
			break
		}
	}
	if !validLen {
		expected := intJoin(rule.LocalLengths, " or ")
		return fmt.Sprintf("Invalid %s number: expected %s digits after +%s, got %d", country, expected, cc, len(local))
	}

	// Check mobile starting digits (if rules exist for this country)
	if len(rule.MobileStartDigits) > 0 && len(local) > 0 {
		validPrefix := false
		for _, prefix := range rule.MobileStartDigits {
			if strings.HasPrefix(local, prefix) {
				validPrefix = true
				break
			}
		}
		if !validPrefix {
			return fmt.Sprintf("Invalid %s mobile number: after +%s must start with %s", country, cc, strings.Join(rule.MobileStartDigits, ", "))
		}
	}

	return ""
}

// intJoin converts a slice of ints to a string joined by sep.
func intJoin(vals []int, sep string) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}
