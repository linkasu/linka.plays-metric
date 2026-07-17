package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"time"
)

const (
	WriterTimestampHeader = "X-Linka-Timestamp"
	WriterBodySHAHeader   = "X-Linka-Body-SHA256"
	WriterSignatureHeader = "X-Linka-Signature"
)

func SignWriterRequest(secret, body []byte, timestamp time.Time) (string, string, string) {
	ts := strconv.FormatInt(timestamp.UTC().Unix(), 10)
	digest := sha256.Sum256(body)
	bodySHA := hex.EncodeToString(digest[:])
	mac := hmac.New(sha256.New, secret)
	_, _ = io.WriteString(mac, ts+"\n"+bodySHA)
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return ts, bodySHA, signature
}

func VerifyWriterRequest(secret, body []byte, now time.Time, maxSkew time.Duration, timestamp, bodySHA, signature string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid writer timestamp")
	}
	requestTime := time.Unix(ts, 0)
	delta := now.Sub(requestTime)
	if delta < -maxSkew || delta > maxSkew {
		return errors.New("writer timestamp outside allowed skew")
	}
	digest := sha256.Sum256(body)
	providedDigest, err := hex.DecodeString(bodySHA)
	if err != nil || !hmac.Equal(providedDigest, digest[:]) {
		return errors.New("invalid writer body digest")
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return errors.New("invalid writer signature")
	}
	_, _, expectedSignature := SignWriterRequest(secret, body, requestTime)
	expected, _ := base64.RawURLEncoding.DecodeString(expectedSignature)
	if !hmac.Equal(providedSignature, expected) {
		return errors.New("invalid writer signature")
	}
	return nil
}
