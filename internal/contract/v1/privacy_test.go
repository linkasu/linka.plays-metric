package v1

import (
	"fmt"
	"testing"
	"time"
)

func TestPublicPrivacyRequestBindsInstallationOutsideBody(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	installationID := "10000000-0000-4000-8000-000000000001"
	body := []byte(fmt.Sprintf(`{"schema_version":1,"request_id":"20000000-0000-4000-8000-000000000002","action":"delete","requested_at":%q}`, now.Format(time.RFC3339)))
	request, err := ParsePublicPrivacyRequest(body, installationID, now)
	if err != nil {
		t.Fatal(err)
	}
	if request.InstallationID != installationID {
		t.Fatalf("installation ID = %s", request.InstallationID)
	}
}

func TestPublicPrivacyRequestRejectsSpoofedInstallation(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	body := []byte(fmt.Sprintf(`{"schema_version":1,"request_id":"20000000-0000-4000-8000-000000000002","installation_id":"30000000-0000-4000-8000-000000000003","action":"delete","requested_at":%q}`, now.Format(time.RFC3339)))
	if _, err := ParsePublicPrivacyRequest(body, "10000000-0000-4000-8000-000000000001", now); err == nil {
		t.Fatal("public request accepted installation_id from body")
	}
}
