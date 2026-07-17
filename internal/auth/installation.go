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
	secret []byte
	now    func() time.Time
	random io.Reader
}

func NewInstallationTokens(secret []byte) (*InstallationTokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("installation HMAC secret must contain at least 32 bytes")
	}
	return &InstallationTokens{
		secret: append([]byte(nil), secret...),
		now:    time.Now,
		random: rand.Reader,
	}, nil
}

func (t *InstallationTokens) Issue() (InstallationClaims, string, error) {
	id, err := uuid.NewRandomFromReader(t.random)
	if err != nil {
		return InstallationClaims{}, "", fmt.Errorf("generate installation ID: %w", err)
	}
	claims := InstallationClaims{
		InstallationID: id.String(),
		IssuedAt:       t.now().UTC().Truncate(time.Second),
	}
	issuedAt := strconv.FormatInt(claims.IssuedAt.Unix(), 10)
	signature := t.sign(claims.InstallationID, issuedAt)
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
	expected := t.sign(parts[1], parts[2])
	if !hmac.Equal(provided, expected) {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	issuedAt := time.Unix(issuedUnix, 0).UTC()
	if issuedAt.After(t.now().Add(5 * time.Minute)) {
		return InstallationClaims{}, errors.New("invalid installation token")
	}
	return InstallationClaims{InstallationID: parts[1], IssuedAt: issuedAt}, nil
}

func (t *InstallationTokens) sign(installationID, issuedAt string) []byte {
	mac := hmac.New(sha256.New, t.secret)
	_, _ = io.WriteString(mac, installationTokenVersion+"\n"+installationID+"\n"+issuedAt)
	return mac.Sum(nil)
}
