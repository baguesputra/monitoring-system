//go:build windows

package main

import "testing"

func TestRIDFromSID(t *testing.T) {
	cases := []struct {
		sid  string
		want int
	}{
		{"S-1-5-21-1234567890-1234567890-1234567890-1000", 1000},
		{"S-1-5-21-1234567890-1234567890-1234567890-1001", 1001},
		{"S-1-5-21-1234567890-1234567890-1234567890-500", 500},
		{"S-1-5-18", 18},
		{"", -1},
		{"not-a-sid", -1},
		{"S-1-5-21-abc-1000", 1000},
	}
	for _, c := range cases {
		if got := ridFromSID(c.sid); got != c.want {
			t.Errorf("ridFromSID(%q) = %d, want %d", c.sid, got, c.want)
		}
	}
}

func TestExtractWMIKeyValue(t *testing.T) {
	ref := `\\.\root\cimv2:Win32_Account.Domain="DESKTOP-92PUGLA",Name="IT 10"`
	if got := extractWMIKeyValue(ref, "Name"); got != "IT 10" {
		t.Errorf("Name = %q, want %q", got, "IT 10")
	}
	if got := extractWMIKeyValue(ref, "Domain"); got != "DESKTOP-92PUGLA" {
		t.Errorf("Domain = %q, want %q", got, "DESKTOP-92PUGLA")
	}
	if got := extractWMIKeyValue(ref, "Missing"); got != "" {
		t.Errorf("Missing = %q, want empty", got)
	}
	if got := extractWMIKeyValue(`Name="unterminated`, "Name"); got != "" {
		t.Errorf("unterminated = %q, want empty", got)
	}
}

func TestTrimDomainPrefix(t *testing.T) {
	cases := map[string]string{
		`DESKTOP-92PUGLA\IT 10`: "IT 10",
		`IT 10`:                "IT 10",
		`\\HOST\Administrators`: "Administrators",
		``:                     "",
	}
	for in, want := range cases {
		if got := trimDomainPrefix(in); got != want {
			t.Errorf("trimDomainPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}
