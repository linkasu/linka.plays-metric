package v1

import (
	"fmt"
	"strings"
	"testing"
)

const validEventTemplate = `{
  "schema_version": 1,
  "events": [{
    "event_id": "10000000-0000-4000-8000-000000000001",
    "event_name": %q,
    "occurred_at": "2026-07-18T12:00:00.123Z",
    "installation_id": "20000000-0000-4000-8000-000000000002",
    "app_session_id": "30000000-0000-4000-8000-000000000003",
    "app": {"version":"1.2.3","build":"42","platform":"linux","os_version":"6.8","locale":"ru-RU"},
    "properties": %s
  }]
}`

func TestParseBatchAcceptsTypedEvent(t *testing.T) {
	body := []byte(fmt.Sprintf(validEventTemplate, "page_viewed", `{"page":"settings"}`))
	batch, err := ParseBatch(body)
	if err != nil {
		t.Fatal(err)
	}
	if got := *batch.Events[0].Dimensions.Page; got != "settings" {
		t.Fatalf("page = %q", got)
	}
}

func TestParseBatchRejectsPrivateOrUnknownProperties(t *testing.T) {
	tests := []struct {
		name       string
		eventName  string
		properties string
	}{
		{"raw x", "page_viewed", `{"page":"game","x":10}`},
		{"raw y", "page_viewed", `{"page":"game","y":10}`},
		{"text", "page_viewed", `{"page":"game","text":"child input"}`},
		{"phrase", "page_viewed", `{"page":"game","phrase":"spoken"}`},
		{"expected", "success", `{"game_id":"letters","level_index":1,"input_method":"gaze","expected":"a"}`},
		{"actual", "mistake", `{"game_id":"letters","level_index":1,"input_method":"gaze","actual":"b"}`},
		{"target id", "target_clicked", `{"game_id":"letters","level_index":1,"target_kind":"letter","input_method":"gaze","targetId":"secret"}`},
		{"path", "error", `{"fingerprint":"abcdef0123456789","component":"renderer","path":"/Users/name"}`},
		{"stack", "error", `{"fingerprint":"abcdef0123456789","component":"renderer","stack":"trace"}`},
		{"message", "error", `{"fingerprint":"abcdef0123456789","component":"renderer","message":"details"}`},
		{"fingerprint text", "error", `{"fingerprint":"renderer.crash","component":"renderer"}`},
		{"missing level", "target_clicked", `{"game_id":"letters","target_kind":"letter","input_method":"gaze"}`},
		{"unsafe value", "page_viewed", `{"page":"/settings/private"}`},
		{"unknown event", "arbitrary_event", `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(fmt.Sprintf(validEventTemplate, test.eventName, test.properties))
			if _, err := ParseBatch(body); err == nil {
				t.Fatal("private or unknown data was accepted")
			}
		})
	}
}

func TestParseBatchRejectsMoreThanFiveHundredRecords(t *testing.T) {
	event := fmt.Sprintf(`{
    "event_id":"10000000-0000-4000-8000-%012d",
    "event_name":"app_started",
    "occurred_at":"2026-07-18T12:00:00Z",
    "installation_id":"20000000-0000-4000-8000-000000000002",
    "app_session_id":"30000000-0000-4000-8000-000000000003",
    "app":{"version":"1","build":"1","platform":"linux","os_version":"1","locale":"ru"},
    "properties":{}
  }`, 1)
	body := `{"schema_version":1,"events":[` + strings.TrimSuffix(strings.Repeat(event+",", 501), ",") + `]}`
	if _, err := ParseBatch([]byte(body)); err == nil {
		t.Fatal("oversized batch was accepted")
	}
}
