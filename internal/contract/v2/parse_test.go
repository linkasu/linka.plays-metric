package v2

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

const testOpaqueKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseBatchAcceptsOneTypedStream(t *testing.T) {
	body := validCommonBatch()
	batch, err := ParseBatch([]byte(body), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if batch.RecordCount() != 1 || len(batch.CommonRecords) != 1 || len(batch.TechnicalRecords) != 0 || len(batch.PlaysRecords) != 0 {
		t.Fatalf("unexpected typed batch: %+v", batch)
	}
}

func TestParseBatchRejectsUnsafeJSONAndEnums(t *testing.T) {
	tests := map[string]string{
		"duplicate": strings.Replace(validCommonBatch(), `"kind":"app_started"`, `"kind":"app_started","kind":"app_closed"`, 1),
		"unknown":   strings.Replace(validCommonBatch(), `"kind":"app_started"`, `"kind":"arbitrary"`, 1),
		"field":     strings.Replace(validCommonBatch(), `"kind":"app_started"`, `"kind":"app_started","text":"private"`, 1),
		"trailing":  validCommonBatch() + `{}`,
		"stream":    strings.Replace(validCommonBatch(), `"stream":"common"`, `"stream":"raw"`, 1),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseBatch([]byte(body), testNow); err == nil {
				t.Fatal("unsafe batch was accepted")
			}
		})
	}
}

func TestParseBatchRejectsTimestampAndRangeViolations(t *testing.T) {
	tests := []string{
		strings.Replace(validCommonBatch(), "2026-07-18T12:00:00.000Z", "2026-07-18T12:00:00.0001Z", 1),
		strings.Replace(validCommonBatch(), "2026-07-18T12:00:00.000Z", "2019-07-18T12:00:00.000Z", 1),
		strings.Replace(validCommonBatch(), "2026-07-18T11:59:00.000Z", "2026-07-18T12:06:00.000Z", 1),
	}
	for _, body := range tests {
		if _, err := ParseBatch([]byte(body), testNow); err == nil {
			t.Fatal("invalid timestamp was accepted")
		}
	}
}

func TestParseBatchRejectsDuplicateRecordIDs(t *testing.T) {
	duplicate := `,{
    "record_id":"20000000-0000-4000-8000-000000000002",
    "occurred_at":"2026-07-18T11:59:00.000Z",
    "kind":"app_started",
    "app_session_id":"40000000-0000-4000-8000-000000000004",
    "app":{"version":"1.2.3","build":"42","platform":"linux","os_version":"6.8","locale":"ru-RU"}
  }`
	body := strings.Replace(validCommonBatch(), "}]", "}"+duplicate+"]", 1)
	if _, err := ParseBatch([]byte(body), testNow); err == nil {
		t.Fatal("duplicate record_id was accepted")
	}
}

func TestParsePrivacyRequestAndIdempotency(t *testing.T) {
	body := fmt.Sprintf(`{"schema_version":2,"request_id":"10000000-0000-4000-8000-000000000001","scope":{"product":"linka-plays","subject_key":%q},"action":"delete","requested_at":"2026-07-18T12:00:00Z"}`, testOpaqueKey)
	request, err := ParsePrivacyRequest([]byte(body), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if request.Action != PrivacyDelete {
		t.Fatalf("action = %q", request.Action)
	}
	if err := ValidateIdempotencyKey(request.RequestID, request.RequestID); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdempotencyKey("20000000-0000-4000-8000-000000000002", request.RequestID); err == nil {
		t.Fatal("mismatched idempotency key was accepted")
	}
}

func validCommonBatch() string {
	return fmt.Sprintf(`{
  "schema_version":2,
  "batch_id":"10000000-0000-4000-8000-000000000001",
  "scope":{"product":"linka-plays","subject_key":%q},
  "stream":"common",
  "sent_at":"2026-07-18T12:00:00.000Z",
  "records":[{
    "record_id":"20000000-0000-4000-8000-000000000002",
    "occurred_at":"2026-07-18T11:59:00.000Z",
    "kind":"app_started",
    "app_session_id":"30000000-0000-4000-8000-000000000003",
    "app":{"version":"1.2.3","build":"42","platform":"linux","os_version":"6.8","locale":"ru-RU"}
  }]
}`, testOpaqueKey)
}
