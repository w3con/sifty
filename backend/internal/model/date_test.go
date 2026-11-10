package model

import "testing"

func TestNormalizePublishedAt(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"date only keeps calendar day", "2025-07-18", "2025-07-18"},
		{"iso milliseconds", "2025-07-18T12:34:56.000Z", "2025-07-18T12:34:56Z"},
		{"space and zone", "2025-07-18 12:34:56 +02:00", "2025-07-18T10:34:56Z"},
		{"web3 stored space no-separator zone", "2026-10-28 05:10:29+07:00", "2026-10-27T22:10:29Z"},
		{"rfc feed date", "Fri, 18 Jul 2025 12:34:56 +0200", "2025-07-18T10:34:56Z"},
		{"empty", "", ""},
		{"invalid", "not-a-date", ""},
		{"future", "2999-01-01", ""},
	}
	for _, tc := range cases {
		if got := NormalizePublishedAt(tc.in); got != tc.want {
			t.Errorf("%s: NormalizePublishedAt(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}
