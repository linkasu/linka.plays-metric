package writer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/linkasu/linka.plays-metric/internal/auth"
	v1 "github.com/linkasu/linka.plays-metric/internal/contract/v1"
)

type fakeStore struct {
	inserted int
	err      error
}

func (s *fakeStore) Ping(context.Context) error {
	return s.err
}

func (s *fakeStore) Insert(_ context.Context, batch v1.ValidatedBatch) error {
	s.inserted += len(batch.Events) + len(batch.SessionSummaries)
	return s.err
}

func TestWriterVerifiesAndInsertsBatch(t *testing.T) {
	store := &fakeStore{}
	secret := []byte(strings.Repeat("w", 32))
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	body := writerValidBatch()
	timestamp, bodySHA, signature := auth.SignWriterRequest(secret, body, now)
	request := httptest.NewRequest(http.MethodPost, "/internal/v1/events", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(auth.WriterTimestampHeader, timestamp)
	request.Header.Set(auth.WriterBodySHAHeader, bodySHA)
	request.Header.Set(auth.WriterSignatureHeader, signature)
	response := httptest.NewRecorder()
	server := &Server{store: store, secret: secret, logger: writerTestLogger(), maxSkew: 5 * time.Minute, now: func() time.Time { return now }}

	server.events(response, request)

	if response.Code != http.StatusAccepted || store.inserted != 1 {
		t.Fatalf("status = %d, inserted = %d, body = %s", response.Code, store.inserted, response.Body.String())
	}
}

func TestWriterRejectsInvalidSignatureWithoutInsert(t *testing.T) {
	store := &fakeStore{}
	request := httptest.NewRequest(http.MethodPost, "/internal/v1/events", bytes.NewReader(writerValidBatch()))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler := NewServer(store, []byte(strings.Repeat("w", 32)), writerTestLogger(), 5*time.Minute)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized || store.inserted != 0 {
		t.Fatalf("status = %d, inserted = %d", response.Code, store.inserted)
	}
}

func writerValidBatch() []byte {
	return []byte(fmt.Sprintf(`{
  "schema_version":1,
  "events":[{
    "event_id":"10000000-0000-4000-8000-000000000001",
    "event_name":"app_started",
    "occurred_at":"2026-07-18T12:00:00.123Z",
    "installation_id":"20000000-0000-4000-8000-000000000002",
    "app_session_id":"30000000-0000-4000-8000-000000000003",
    "app":{"version":"1","build":"1","platform":"linux","os_version":"1","locale":"ru"},
    "properties":{}
  }]
}`))
}

func writerTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
