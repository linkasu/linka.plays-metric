package auth

import (
	"strings"
	"testing"
	"time"
)

func TestWriterRequestSignature(t *testing.T) {
	secret := []byte(strings.Repeat("w", 32))
	body := []byte(`{"schema_version":1}`)
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	timestamp, bodySHA, signature := SignWriterRequest(secret, body, now)

	if err := VerifyWriterRequest(secret, body, now.Add(time.Minute), 5*time.Minute, timestamp, bodySHA, signature); err != nil {
		t.Fatal(err)
	}
	if err := VerifyWriterRequest(secret, append(body, ' '), now, 5*time.Minute, timestamp, bodySHA, signature); err == nil {
		t.Fatal("changed body was accepted")
	}
	if err := VerifyWriterRequest(secret, body, now.Add(6*time.Minute), 5*time.Minute, timestamp, bodySHA, signature); err == nil {
		t.Fatal("stale signature was accepted")
	}
}
