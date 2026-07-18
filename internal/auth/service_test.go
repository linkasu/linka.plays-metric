package auth

import (
	"strings"
	"testing"
	"time"
)

func TestServiceHMACBindsCompleteRequest(t *testing.T) {
	key := ServiceKey{ID: "collector-2026-07", Secret: []byte(strings.Repeat("k", 32))}
	signer, err := NewServiceSigner(key, "collector")
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := NewServiceVerifier(key, nil, "collector", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return now }
	verifier.now = func() time.Time { return now }
	body := []byte(`{"schema_version":2}`)
	headers, err := signer.Sign("POST", "/internal/v2/batches", "10000000-0000-4000-8000-000000000001", body)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify("POST", "/internal/v2/batches", body, headers); err != nil {
		t.Fatal(err)
	}
	mutations := []func(ServiceHeaders) ServiceHeaders{
		func(h ServiceHeaders) ServiceHeaders { h.Caller = "privacy-worker"; return h },
		func(h ServiceHeaders) ServiceHeaders { h.KeyID = "other"; return h },
		func(h ServiceHeaders) ServiceHeaders { h.RequestID = "other"; return h },
	}
	for _, mutate := range mutations {
		if err := verifier.Verify("POST", "/internal/v2/batches", body, mutate(headers)); err == nil {
			t.Fatal("mutated service request was accepted")
		}
	}
	if err := verifier.Verify("PUT", "/internal/v2/batches", body, headers); err == nil {
		t.Fatal("changed method was accepted")
	}
	if err := verifier.Verify("POST", "/internal/v2/privacy/requests", body, headers); err == nil {
		t.Fatal("changed path was accepted")
	}
	if err := verifier.Verify("POST", "/internal/v2/batches", append(body, ' '), headers); err == nil {
		t.Fatal("changed body was accepted")
	}
}

func TestServiceVerifierAcceptsPreviousKey(t *testing.T) {
	oldKey := ServiceKey{ID: "old", Secret: []byte(strings.Repeat("o", 32))}
	signer, _ := NewServiceSigner(oldKey, "collector")
	newKey := ServiceKey{ID: "new", Secret: []byte(strings.Repeat("n", 32))}
	verifier, err := NewServiceVerifier(newKey, &oldKey, "collector", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	signer.now = func() time.Time { return now }
	verifier.now = func() time.Time { return now }
	headers, _ := signer.Sign("POST", "/internal/v2/batches", "request", nil)
	if err := verifier.Verify("POST", "/internal/v2/batches", nil, headers); err != nil {
		t.Fatal(err)
	}
}
