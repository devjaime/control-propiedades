package httpserver

import "testing"

func TestDocumentTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		allowed bool
	}{
		{name: "upload to review", current: "uploaded", next: "pending_review", allowed: true},
		{name: "review to verified", current: "pending_review", next: "verified", allowed: true},
		{name: "rejected back to review", current: "rejected", next: "pending_review", allowed: true},
		{name: "verified cannot return to review", current: "verified", next: "pending_review", allowed: false},
		{name: "archived cannot be verified", current: "archived", next: "verified", allowed: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validDocumentTransition(test.current, test.next); got != test.allowed {
				t.Fatalf("validDocumentTransition(%q, %q) = %v, want %v", test.current, test.next, got, test.allowed)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	tags := normalizeTagSlice([]string{" Legal ", "2026", "legal", "", "Vigente"})
	want := []string{"legal", "2026", "vigente"}
	if len(tags) != len(want) {
		t.Fatalf("normalizeTagSlice() = %v, want %v", tags, want)
	}
	for index := range want {
		if tags[index] != want[index] {
			t.Fatalf("normalizeTagSlice() = %v, want %v", tags, want)
		}
	}
}

func TestDirectUploadHashValidation(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if !isLowerHex(valid) {
		t.Fatal("expected lowercase SHA-256 to be valid")
	}
	if isLowerHex("ABCDEF") || isLowerHex("not-a-hash") {
		t.Fatal("expected non-lowercase hexadecimal values to be rejected")
	}
}

func TestMetadataValueIsCaseInsensitive(t *testing.T) {
	metadata := map[string]string{"X-Amz-Meta-Sha256": " abc123 "}
	if got := metadataValue(metadata, "x-amz-meta-sha256"); got != "abc123" {
		t.Fatalf("metadataValue() = %q, want abc123", got)
	}
}
