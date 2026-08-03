package httpserver

import "testing"

func TestRUTAccessCode(t *testing.T) {
	tests := []struct {
		identifier string
		want       string
		ok         bool
	}{
		{"11.111.111-1", "1111", true},
		{"12.345.678-5", "5678", true},
		{"12.345.678-K", "5678", true},
		{"123", "", false},
	}
	for _, test := range tests {
		got, ok := rutAccessCode(test.identifier)
		if ok != test.ok || got != test.want {
			t.Errorf("rutAccessCode(%q) = %q, %t; want %q, %t", test.identifier, got, ok, test.want, test.ok)
		}
	}
}
