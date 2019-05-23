package turtlequote

import (
	"os/exec"
	"strings"
	"testing"
	"testing/quick"
	"unicode"
)

func TestEscape(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{"東方", "東方"},
		{"\"'", `"\"'"`},
		{"\\", "\"\\\\\""},
		{"spaces only", "'spaces only'"},
		{"some\ttabs", "\"some\\ttabs\""},
		{"💩", "💩"},
		{"\u202eRTL", `"\u{202e}RTL"`},
		{"no\u202bspace", `"no\u{202b}space"`},
		{"cash $ money $$ \t", "\"cash \\$ money \\$\\$ \\t\""},
		{"back ` tick `` \t", "\"back \\` tick \\`\\` \\t\""},
		{
			"\u0007\u0008\u000b\u000c\u000a\u000d\u0009\u001b\u001b\u005c\u0027\u0022",
			"\"\\a\\b\\v\\f\\n\\r\\t\\e\\e\\\\'\\\"\"",
		},
	}

	for _, tt := range testCases {
		out := Escape(tt.in)
		if tt.out != out {
			t.Fatalf("expected Escape(%q) == %q; got %q", tt.in, tt.out, out)
		}
	}
}

func TestUnescape(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{"\"\\u{6771}\\u{65b9}\"", "東方"},
		{"東方", "東方"},
		{"\"\\\\\"'\"\"'", "\\\"\""},
		{"'\"'", "\""},
		{"'\"'", "\""},
		{"\"\U0008e098\U000e2c6e\U0003f4ed\U00093d61\\u{000f6780}\"", "\U0008e098\U000e2c6e\U0003f4ed\U00093d61\U000f6780"},
		{
			"\"\\a\\b\\v\\f\\n\\r\\t\\e\\E\\\\\\'\\\"\\u{09}\\$\\`\"",
			"\u0007\u0008\u000b\u000c\u000a\u000d\u0009\u001b\u001b\u005c\u0027\u0022\u0009$`",
		},
		{
			"\"\U000783ba䂰\U00083c8f\"",
			"\U000783ba䂰\U00083c8f",
		},
		{
			"\"\\u{0010dbcb}'𡗽\U000b3219\U00056b65\\u{00108779}𮡚\U000d241a\U00014c85\U000783ba䂰\U00083c8f\"",
			"\U0010dbcb'𡗽\U000b3219\U00056b65\U00108779𮡚\U000d241a\U00014c85\U000783ba䂰\U00083c8f",
		},
		{
			"\"\U0006e591\U0003df25\\u{001083c9}\U000377ec𢓿\\u{000fb03d}\U000d56e8𑫗\U0003fdee\U0003bc56\U000d0dd5\U00099238\\u{00106c95}\U000dcbd5\U000d361a\\u{00101f92}\U00031c98歉\U000dc6e7\U000e7b1a\U0005f5d8\U000b6d63𭊯\U000c35ec\U0007f4b2\U00060022\U0009a3ec\U000678d7\\u{000f15bc}\\u{0010dbcb}'𡗽\U000b3219\U00056b65\\u{00108779}𮡚\U000d241a\U00014c85\U000783ba䂰\U00083c8f\"",
			"\U0006e591\U0003df25\U001083c9\U000377ec𢓿\U000fb03d\U000d56e8𑫗\U0003fdee\U0003bc56\U000d0dd5\U00099238\U00106c95\U000dcbd5\U000d361a\U00101f92\U00031c98歉\U000dc6e7\U000e7b1a\U0005f5d8\U000b6d63𭊯\U000c35ec\U0007f4b2\U00060022\U0009a3ec\U000678d7\U000f15bc\U0010dbcb'𡗽\U000b3219\U00056b65\U00108779𮡚\U000d241a\U00014c85\U000783ba䂰\U00083c8f",
		},
	}

	for _, tt := range testCases {
		out, err := Unescape(tt.in)
		if err != nil {
			t.Fatalf("expected nil error for Unescape(%q): %v", tt.in, err)
		}
		if tt.out != out {
			t.Fatalf("expected Unescape(%q) == %q; got %q", tt.in, tt.out, out)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	f := func(s string) bool {
		esc := Escape(s)
		rt, err := Unescape(esc)
		if err != nil {
			t.Fatalf("could not rt:\n\tEscape(%q) == \n\t       %q; Unescape gave error %v", s, esc, err)
		}
		return rt == s
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestShellRoundTrips(t *testing.T) {
	f := func(s string) bool {
		s = strings.Map(func(r rune) rune {
			// trim weird unicode; the shell can't unescape \u{foo} correctly
			if !unicode.IsPrint(r) {
				return 'x'
			}
			return r
		}, s)
		t.Logf(s)
		esc := Escape(s)
		output, err := exec.Command("sh", "-c", "echo "+esc).CombinedOutput()
		if err != nil {
			t.Errorf("error on %q (output %q): %v", s, string(output), err)
			return false
		}
		if len(output) == 0 {
			t.Error("zero length output; expected newline at end")
			return false
		}
		outStr := string(output[0 : len(output)-1])
		if outStr != s {
			t.Errorf("echo gave %q, not %q", outStr, s)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestShellRoundTripCases(t *testing.T) {
	s := "xxxx&xxxxxxxxxx𐆚xxxxxxxxxxxxxxxx侀xxxxx"
	esc := Escape(s)
	t.Logf("escaped: %v", esc)
	output, err := exec.Command("sh", "-c", "echo "+esc).CombinedOutput()
	if err != nil {
		t.Errorf("error on %q (output %q): %v", s, string(output), err)
	}
	if len(output) == 0 {
		t.Error("zero length output; expected newline at end")
	}
	outStr := string(output[0 : len(output)-1])
	if outStr != s {
		t.Errorf("echo gave %q, not %q", outStr, s)
	}
}
