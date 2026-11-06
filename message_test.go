package kwtsms

import (
	"strings"
	"testing"
)

func TestCleanMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain English", "Hello World", "Hello World"},
		{"plain Arabic", "مرحبا بالعالم", "مرحبا بالعالم"},
		{"Arabic digits converted", "الرمز ١٢٣٤٥٦", "الرمز 123456"},
		{"Persian digits converted", "۱۲۳۴۵۶", "123456"},
		{"mixed Arabic and Latin digits", "Code: ١٢٣ and 456", "Code: 123 and 456"},
		{"emoji stripped", "Hello 😀 World 🎉", "Hello  World "},
		{"flag emoji stripped", "🇰🇼 Kuwait", " Kuwait"},
		{"BOM stripped", "\uFEFFHello", "Hello"},
		{"zero-width space stripped", "Hello\u200BWorld", "HelloWorld"},
		{"soft hyphen stripped", "Hello\u00ADWorld", "HelloWorld"},
		{"zero-width joiner stripped", "Hello\u200DWorld", "HelloWorld"},
		{"HTML tags stripped", "<b>Hello</b> <i>World</i>", "Hello World"},
		{"HTML script stripped", "<script>alert('xss')</script>Hello", "alert('xss')Hello"},
		{"multi-line HTML", "<div\nclass='x'>content</div>", "content"},
		{"newlines preserved", "Line1\nLine2\nLine3", "Line1\nLine2\nLine3"},
		{"tabs preserved", "Col1\tCol2", "Col1\tCol2"},
		{"C0 control stripped", "Hello\x01\x02World", "HelloWorld"},
		{"DEL stripped", "Hello\x7FWorld", "HelloWorld"},
		{"C1 control stripped", "Hello\u0080\u009FWorld", "HelloWorld"},
		{"LTR mark stripped", "Hello\u200EWorld", "HelloWorld"},
		{"RTL mark stripped", "Hello\u200FWorld", "HelloWorld"},
		{"directional formatting stripped", "Hello\u202AWorld\u202C", "HelloWorld"},
		{"variation selector stripped", "Hello\uFE0FWorld", "HelloWorld"},
		{"keycap stripped", "1\u20E32\u20E3", "12"},
		{"empty after cleaning", "😀🎉🇰🇼", ""},
		{"only emojis and spaces", "  😀  🎉  ", "      "},
		{"word joiner stripped", "Hello\u2060World", "HelloWorld"},
		{"object replacement stripped", "Hello\uFFFCWorld", "HelloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanMessage(tt.input)
			if got != tt.want {
				t.Errorf("CleanMessage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanMessagePreservesArabicLetters(t *testing.T) {
	arabic := "مرحبا بالعالم هذا نص عربي"
	got := CleanMessage(arabic)
	if got != arabic {
		t.Errorf("CleanMessage should preserve Arabic text.\nGot:  %q\nWant: %q", got, arabic)
	}
}

func TestCleanMessageEmptyInput(t *testing.T) {
	if got := CleanMessage(""); got != "" {
		t.Errorf("CleanMessage(\"\") = %q, want \"\"", got)
	}
}

func TestCleanMessageLongText(t *testing.T) {
	long := strings.Repeat("Hello World. ", 100)
	got := CleanMessage(long)
	if got != long {
		t.Error("CleanMessage should not alter clean long text")
	}
}
