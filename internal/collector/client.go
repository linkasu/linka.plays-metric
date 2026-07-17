package collector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linkasu/linka.plays-metric/internal/auth"
)

type HTTPWriter struct {
	endpoint string
	secret   []byte
	client   *http.Client
	now      func() time.Time
}

func NewHTTPWriter(baseURL string, secret []byte, timeout time.Duration) (*HTTPWriter, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("invalid WRITER_URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/internal/v1/events"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return &HTTPWriter{
		endpoint: parsed.String(),
		secret:   append([]byte(nil), secret...),
		client:   &http.Client{Timeout: timeout},
		now:      time.Now,
	}, nil
}

func (w *HTTPWriter) Write(ctx context.Context, body []byte) error {
	timestamp, bodySHA, signature := auth.SignWriterRequest(w.secret, body, w.now())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create writer request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(auth.WriterTimestampHeader, timestamp)
	request.Header.Set(auth.WriterBodySHAHeader, bodySHA)
	request.Header.Set(auth.WriterSignatureHeader, signature)
	response, err := w.client.Do(request)
	if err != nil {
		return fmt.Errorf("call writer: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("writer returned HTTP %d", response.StatusCode)
	}
	return nil
}
