package auth

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestInstallationTokenIssueAndVerify(t *testing.T) {
	manager, err := NewInstallationTokens([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }
	manager.random = bytes.NewReader([]byte("0123456789abcdef"))

	issued, token, err := manager.Issue()
	if err != nil {
		t.Fatal(err)
	}
	verified, err := manager.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if verified != issued {
		t.Fatalf("verified claims differ: got %+v, want %+v", verified, issued)
	}
}

func TestInstallationTokenRejectsTampering(t *testing.T) {
	manager, err := NewInstallationTokens([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := manager.Issue()
	if err != nil {
		t.Fatal(err)
	}
	tampered := token[:len(token)-1] + "A"
	if tampered == token {
		tampered = token[:len(token)-1] + "B"
	}
	if _, err := manager.Verify(tampered); err == nil {
		t.Fatal("tampered token was accepted")
	}
}

func TestInstallationTokenSecretLength(t *testing.T) {
	if _, err := NewInstallationTokens([]byte("short")); err == nil {
		t.Fatal("short secret was accepted")
	}
}
