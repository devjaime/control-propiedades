package objectstorage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPresignPutPreservesSupabaseEndpointPath(t *testing.T) {
	storage, err := New(
		"https://project-ref.storage.supabase.co/storage/v1/s3",
		"us-east-1",
		"test-access-key",
		"test-secret-key",
		"control-propiedades",
		true,
	)
	if err != nil {
		t.Fatal(err)
	}

	signedURL, headers, err := storage.PresignPut(context.Background(), "documents/test.pdf", "application/pdf", strings.Repeat("a", 64), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/storage/v1/s3/control-propiedades/documents/test.pdf" {
		t.Fatalf("unexpected signed path: %s", parsed.Path)
	}
	if headers["X-Amz-Meta-Sha256"] == "" {
		t.Fatal("missing signed checksum metadata header")
	}
}

func TestNewNormalizesLocalEndpoint(t *testing.T) {
	storage, err := New("localhost:9000", "us-east-1", "access", "secret", "control-propiedades", false)
	if err != nil {
		t.Fatal(err)
	}
	signedURL, _, err := storage.PresignPut(context.Background(), "test.txt", "text/plain", strings.Repeat("b", 64), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "http" || parsed.Host != "localhost:9000" {
		t.Fatalf("unexpected local endpoint: %s", parsed.String())
	}
}
