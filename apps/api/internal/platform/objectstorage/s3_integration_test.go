package objectstorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSupabaseMultipartRoundTrip(t *testing.T) {
	if os.Getenv("STORAGE_INTEGRATION") != "1" {
		t.Skip("set STORAGE_INTEGRATION=1 to run against configured S3 storage")
	}
	endpoint := requiredTestEnv(t, "TARGET_STORAGE_ENDPOINT")
	region := requiredTestEnv(t, "TARGET_STORAGE_REGION")
	accessKey := requiredTestEnv(t, "TARGET_STORAGE_ACCESS_KEY")
	secretKey := requiredTestEnv(t, "TARGET_STORAGE_SECRET_KEY")
	bucket := requiredTestEnv(t, "TARGET_STORAGE_BUCKET")

	storage, err := New(endpoint, region, accessKey, secretKey, bucket, true)
	if err != nil {
		t.Fatal(err)
	}
	first := bytes.Repeat([]byte("a"), int(multipartPartSize))
	second := bytes.Repeat([]byte("b"), 1024*1024)
	content := append(first, second...)
	digest := sha256.Sum256(content)
	checksum := hex.EncodeToString(digest[:])
	key := fmt.Sprintf("integration-tests/multipart-%d.bin", time.Now().UnixNano())
	ctx := context.Background()
	uploadID, parts, err := storage.InitiateMultipart(ctx, key, "application/octet-stream", checksum, 2, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = storage.AbortMultipart(context.Background(), key, uploadID)
		_ = storage.Delete(context.Background(), key)
	}()

	etags := make(map[int32]string, 2)
	for partNumber, body := range map[int32][]byte{1: first, 2: second} {
		request, err := http.NewRequestWithContext(ctx, http.MethodPut, parts[partNumber], bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Origin", "https://example.vercel.app")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			t.Fatalf("part %d upload returned HTTP %d", partNumber, response.StatusCode)
		}
		etag := strings.TrimSpace(response.Header.Get("ETag"))
		if etag == "" {
			t.Fatalf("part %d response did not expose ETag", partNumber)
		}
		etags[partNumber] = etag
	}
	if err := storage.CompleteMultipart(ctx, key, uploadID, etags); err != nil {
		t.Fatal(err)
	}
	size, contentType, metadata, err := storage.Stat(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(content)) {
		t.Fatalf("stored size %d, expected %d", size, len(content))
	}
	if contentType != "application/octet-stream" {
		t.Fatalf("stored content type %q", contentType)
	}
	if !strings.EqualFold(metadata["sha256"], checksum) {
		t.Fatalf("stored checksum metadata does not match")
	}
}

func requiredTestEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("missing %s", name)
	}
	return value
}
