package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const installationTokenVersion = "v1"

type InstallationClaims struct {
	InstallationID string
	IssuedAt       time.Time
}

type InstallationTokens struct {
	activeSecret   []byte
	previousSecret []byte
	now            func() time.Time
	random         io.Reader
	maxAge         time.Duration
}

func NewInstallationTokens(secret []byte) (*InstallationTokens, error) {
	return NewInstallationTokensWithMaxAge(secret, 30*24*time.Hour)
}

func NewInstallationTokensWithMaxAge(secret []byte, maxAge time.Duration) (*InstallationTokens, error) {
	return NewInstallationTokensWithKeyring(secret, nil, maxAge)
}

func NewInstallationTokensWithKeyring(activeSecret, previousSecret []byte, maxAge time.Duration) (*InstallationTokens, error) {
	if len(activeSecret) < 32 || (previousSecret != nil && len(previousSecret) < 32) {
		return nil, errors.New("installation HMAC secret must contain at least 32 bytes")
	}
	if maxAge < time.Hour || maxAge > 365*24*time.Hour {
		return nil, errors.New("installation token max age must be between one hour and one year")
	}
	return &InstallationTokens{
		activeSecret:   append([]byte(nil), activeSecret...),
		previousSecret: append([]byte(nil), previousSecret...),
		now:            time.Now,
		random:         rand.Reader,
		maxAge:         maxAge,
	}, nil
}

func (t *InstallationTokens) Issue() (InstallationClaims, string, error) {
	id, err := uuid.NewRandomFromReader(t.random)
	if err != nil {
		return InstallationClaims{}, "", fmt.Errorf("generate installation ID: %w", err)
	}
	return t.issue(id.String())
}

func (t *InstallationTokens) Renew(token string) (InstallationClaims, string, error) {
	claims, err := t.Verify(token)
	if err != nil {
		return InstallationClaims{}, "", err
	}
	return t.issue(claims.InstallationID)
}

func (t *InstallationTokens) issue(installationID string) (InstallationClaims, string, error) {
	claims := InstallationClaims{InstallationID: installationID, IssuedAt: t.now().UTC().Truncate(time.Second)}
	issuedAt := strconv.FormatInt(claims.IssuedAt.Unix(), 10)
	signature := t.sign(t.activeSecret, claims.InstallationID, issuedAt)
	token := strings.Join([]string{
		installationTokenVersion,
		claims.InstallationID,
		issuedAt,
		base64.RawURLEncoding.EncodeToString(signature),
	}, ".")
	return claims, token, nil
}

func (t *InstallationTokens) Verify(token string) (InstallationClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != installationTokenVersion {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	issuedUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	provided, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	valid := hmac.Equal(provided, t.sign(t.activeSecret, parts[1], parts[2]))
	if t.previousSecret != nil {
		valid = hmac.Equal(provided, t.sign(t.previousSecret, parts[1], parts[2])) || valid
	}
	if !valid {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	issuedAt := time.Unix(issuedUnix, 0).UTC()
	if issuedAt.After(t.now().Add(5*time.Minute)) || issuedAt.Before(t.now().Add(-t.maxAge)) {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	return InstallationClaims{InstallationID: parts[1], IssuedAt: issuedAt}, nil
}

func (t *InstallationTokens) sign(secret []byte, installationID, issuedAt string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = io.WriteString(mac, installationTokenVersion+"\n"+installationID+"\n"+issuedAt)
	return mac.Sum(nil)
}
