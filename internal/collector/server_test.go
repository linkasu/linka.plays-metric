package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linkasu/linka.plays-metric/internal/auth"
)

type fakeEventWriter struct {
	body []byte
	err  error
}

func (w *fakeEventWriter) Write(_ context.Context, body []byte) error {
	w.body = append([]byte(nil), body...)
	return w.err
}

func TestEventsForwardsExactBodyAfterWriterSuccess(t *testing.T) {
	writer := &fakeEventWriter{}
	tokens, err := auth.NewInstallationTokens([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	claims, token, err := tokens.Issue()
	if err != nil {
		t.Fatal(err)
	}
	body := validBatch(claims.InstallationID)
	request := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	NewServer(writer, tokens, testLogger()).ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !bytes.Equal(writer.body, body) {
		t.Fatal("collector changed the forwarded body")
	}
}

func TestEventsDoesNotSucceedWhenWriterFails(t *testing.T) {
	writer := &fakeEventWriter{err: errors.New("unavailable")}
	tokens, err := auth.NewInstallationTokens([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	claims, token, err := tokens.Issue()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(validBatch(claims.InstallationID)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	NewServer(writer, tokens, testLogger()).ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestInstallationEndpointRejectsUnknownFields(t *testing.T) {
	tokens, err := auth.NewInstallationTokens([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/installations", strings.NewReader(`{"device_id":"forbidden"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	NewServer(&fakeEventWriter{}, tokens, testLogger()).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func validBatch(installationID string) []byte {
	return []byte(fmt.Sprintf(`{
  "schema_version":1,
  "events":[{
    "event_id":"10000000-0000-4000-8000-000000000001",
    "event_name":"app_started",
    "occurred_at":"2026-07-18T12:00:00.123Z",
    "installation_id":%q,
    "app_session_id":"30000000-0000-4000-8000-000000000003",
    "app":{"version":"1.2.3","build":"42","platform":"linux","os_version":"6.8","locale":"ru-RU"},
    "properties":{}
  }]
}`, installationID))
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
