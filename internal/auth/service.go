package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ServiceSignatureVersionHeader = "X-Linka-Signature-Version"
	ServiceKeyIDHeader            = "X-Linka-Key-ID"
	ServiceCallerHeader           = "X-Linka-Caller"
	ServiceRequestIDHeader        = "X-Linka-Request-ID"
	ServiceTimestampHeader        = "X-Linka-Timestamp"
	ServiceBodySHAHeader          = "X-Linka-Body-SHA256"
	ServiceSignatureHeader        = "X-Linka-Signature"
	ServiceSignatureVersion       = "v2"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type ServiceKey struct {
	ID     string
	Secret []byte
}

type ServiceSigner struct {
	key    ServiceKey
	caller string
	now    func() time.Time
}

type ServiceVerifier struct {
	active        ServiceKey
	previous      *ServiceKey
	allowedCaller string
	maxSkew       time.Duration
	now           func() time.Time
}

type ServiceHeaders struct {
	Version   string
	KeyID     string
	Caller    string
	RequestID string
	Timestamp string
	BodySHA   string
	Signature string
}

func NewServiceSigner(key ServiceKey, caller string) (*ServiceSigner, error) {
	if err := validateServiceKey(key); err != nil {
		return nil, err
	}
	if !serviceNamePattern.MatchString(caller) {
		return nil, errors.New("invalid service caller")
	}
	return &ServiceSigner{key: cloneServiceKey(key), caller: caller, now: time.Now}, nil
}

func NewServiceVerifier(active ServiceKey, previous *ServiceKey, allowedCaller string, maxSkew time.Duration) (*ServiceVerifier, error) {
	if err := validateServiceKey(active); err != nil {
		return nil, fmt.Errorf("active service key: %w", err)
	}
	if previous != nil {
		if err := validateServiceKey(*previous); err != nil {
			return nil, fmt.Errorf("previous service key: %w", err)
		}
		if previous.ID == active.ID {
			return nil, errors.New("active and previous service key IDs must differ")
		}
	}
	if !serviceNamePattern.MatchString(allowedCaller) || maxSkew <= 0 {
		return nil, errors.New("invalid service verifier configuration")
	}
	return &ServiceVerifier{
		active: cloneServiceKey(active), previous: cloneServiceKeyPointer(previous), allowedCaller: allowedCaller, maxSkew: maxSkew, now: time.Now,
	}, nil
}

func (s *ServiceSigner) Sign(method, path, requestID string, body []byte) (ServiceHeaders, error) {
	if method == "" || path == "" || requestID == "" || len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
		return ServiceHeaders{}, errors.New("invalid service request metadata")
	}
	timestamp := strconv.FormatInt(s.now().UTC().Unix(), 10)
	digest := sha256.Sum256(body)
	headers := ServiceHeaders{
		Version: ServiceSignatureVersion, KeyID: s.key.ID, Caller: s.caller, RequestID: requestID,
		Timestamp: timestamp, BodySHA: hex.EncodeToString(digest[:]),
	}
	headers.Signature = base64.RawURLEncoding.EncodeToString(serviceMAC(s.key.Secret, method, path, headers))
	return headers, nil
}

func (v *ServiceVerifier) Verify(method, path string, body []byte, headers ServiceHeaders) error {
	if headers.Version != ServiceSignatureVersion || headers.Caller != v.allowedCaller || headers.RequestID == "" || len(headers.RequestID) > 128 || strings.ContainsAny(headers.RequestID, "\r\n") {
		return errors.New("invalid service signature metadata")
	}
	key, ok := v.key(headers.KeyID)
	if !ok {
		return errors.New("unknown service key ID")
	}
	timestamp, err := strconv.ParseInt(headers.Timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid service timestamp")
	}
	delta := v.now().Sub(time.Unix(timestamp, 0))
	if delta < -v.maxSkew || delta > v.maxSkew {
		return errors.New("service timestamp outside allowed skew")
	}
	digest := sha256.Sum256(body)
	providedDigest, err := hex.DecodeString(headers.BodySHA)
	if err != nil || headers.BodySHA != hex.EncodeToString(digest[:]) || !hmac.Equal(providedDigest, digest[:]) {
		return errors.New("invalid service body digest")
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(headers.Signature)
	if err != nil || !hmac.Equal(providedSignature, serviceMAC(key.Secret, method, path, headers)) {
		return errors.New("invalid service signature")
	}
	return nil
}

func ServiceHeadersFromRequest(request *http.Request) ServiceHeaders {
	return ServiceHeaders{
		Version:   request.Header.Get(ServiceSignatureVersionHeader),
		KeyID:     request.Header.Get(ServiceKeyIDHeader),
		Caller:    request.Header.Get(ServiceCallerHeader),
		RequestID: request.Header.Get(ServiceRequestIDHeader),
		Timestamp: request.Header.Get(ServiceTimestampHeader),
		BodySHA:   request.Header.Get(ServiceBodySHAHeader),
		Signature: request.Header.Get(ServiceSignatureHeader),
	}
}

func ApplyServiceHeaders(header http.Header, headers ServiceHeaders) {
	header.Set(ServiceSignatureVersionHeader, headers.Version)
	header.Set(ServiceKeyIDHeader, headers.KeyID)
	header.Set(ServiceCallerHeader, headers.Caller)
	header.Set(ServiceRequestIDHeader, headers.RequestID)
	header.Set(ServiceTimestampHeader, headers.Timestamp)
	header.Set(ServiceBodySHAHeader, headers.BodySHA)
	header.Set(ServiceSignatureHeader, headers.Signature)
}

func serviceMAC(secret []byte, method, path string, headers ServiceHeaders) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = io.WriteString(mac, strings.Join([]string{
		"LINKA-HMAC-V2", headers.Version, strings.ToUpper(method), path, headers.Caller, headers.KeyID,
		headers.RequestID, headers.Timestamp, headers.BodySHA,
	}, "\n"))
	return mac.Sum(nil)
}

func (v *ServiceVerifier) key(id string) (ServiceKey, bool) {
	if id == v.active.ID {
		return v.active, true
	}
	if v.previous != nil && id == v.previous.ID {
		return *v.previous, true
	}
	return ServiceKey{}, false
}

func validateServiceKey(key ServiceKey) error {
	if !serviceNamePattern.MatchString(key.ID) {
		return errors.New("invalid service key ID")
	}
	if len(key.Secret) < 32 {
		return errors.New("service secret must contain at least 32 bytes")
	}
	return nil
}

func cloneServiceKey(key ServiceKey) ServiceKey {
	return ServiceKey{ID: key.ID, Secret: append([]byte(nil), key.Secret...)}
}

func cloneServiceKeyPointer(key *ServiceKey) *ServiceKey {
	if key == nil {
		return nil
	}
	cloned := cloneServiceKey(*key)
	return &cloned
}
