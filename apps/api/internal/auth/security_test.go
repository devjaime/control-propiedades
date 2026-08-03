package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("una-clave-segura")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !ComparePassword(hash, "una-clave-segura") {
		t.Fatal("expected password to match")
	}
	if ComparePassword(hash, "otra-clave-segura") {
		t.Fatal("expected different password not to match")
	}
}

func TestHashAccessCode(t *testing.T) {
	hash, err := HashAccessCode("7406")
	if err != nil {
		t.Fatal(err)
	}
	if !ComparePassword(hash, "7406") {
		t.Fatal("expected access code to match")
	}
	if ComparePassword(hash, "8517") {
		t.Fatal("unexpected access code match")
	}
	for _, invalid := range []string{"406", "17406", "74A6"} {
		if _, err := HashAccessCode(invalid); err == nil {
			t.Fatalf("HashAccessCode(%q) expected error", invalid)
		}
	}
}

func TestSessionTokensAreHashed(t *testing.T) {
	plain, hash, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken() error = %v", err)
	}
	if plain == string(hash) {
		t.Fatal("session token must not be stored in plaintext")
	}
	if got := HashSessionToken(plain); string(got) != string(hash) {
		t.Fatal("session token hash is not stable")
	}
}
