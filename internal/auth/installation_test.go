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

func TestInstallationTokenExpiresAndRenewsWithoutChangingInstallation(t *testing.T) {
	manager, err := NewInstallationTokensWithMaxAge([]byte(strings.Repeat("s", 32)), 2*time.Hour)
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
	now = now.Add(time.Hour)
	renewedClaims, renewedToken, err := manager.Renew(token)
	if err != nil {
		t.Fatal(err)
	}
	if renewedClaims.InstallationID != issued.InstallationID {
		t.Fatalf("installation changed during renewal: got %s, want %s", renewedClaims.InstallationID, issued.InstallationID)
	}

	now = now.Add(90 * time.Minute)
	if _, err := manager.Verify(token); err == nil {
		t.Fatal("expired original token was accepted")
	}
	if _, err := manager.Verify(renewedToken); err != nil {
		t.Fatalf("renewed token was rejected: %v", err)
	}
}

func TestInstallationTokenMaxAgeBounds(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	if _, err := NewInstallationTokensWithMaxAge(secret, time.Minute); err == nil {
		t.Fatal("one-minute max age was accepted")
	}
	if _, err := NewInstallationTokensWithMaxAge(secret, 366*24*time.Hour); err == nil {
		t.Fatal("max age over one year was accepted")
	}
}

func TestInstallationTokenPreviousKeyVerificationAndRenewal(t *testing.T) {
	oldSecret := []byte(strings.Repeat("o", 32))
	newSecret := []byte(strings.Repeat("n", 32))
	oldManager, err := NewInstallationTokensWithKeyring(oldSecret, nil, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	oldManager.now = func() time.Time { return now }
	oldManager.random = bytes.NewReader([]byte("0123456789abcdef"))
	claims, oldToken, err := oldManager.Issue()
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := NewInstallationTokensWithKeyring(newSecret, oldSecret, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rotated.now = func() time.Time { return now.Add(time.Minute) }
	if verified, err := rotated.Verify(oldToken); err != nil || verified.InstallationID != claims.InstallationID {
		t.Fatalf("verify previous-key token: claims=%+v err=%v", verified, err)
	}
	_, renewedToken, err := rotated.Renew(oldToken)
	if err != nil {
		t.Fatal(err)
	}
	newOnly, _ := NewInstallationTokensWithKeyring(newSecret, nil, 24*time.Hour)
	newOnly.now = rotated.now
	if _, err := newOnly.Verify(renewedToken); err != nil {
		t.Fatalf("renewed token was not signed by active key: %v", err)
	}
	if _, err := newOnly.Verify(oldToken); err == nil {
		t.Fatal("old token was accepted after previous key removal")
	}
}
