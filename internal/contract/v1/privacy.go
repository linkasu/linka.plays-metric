package v1

import (
	"errors"
	"time"

	"github.com/linkasu/linka.plays-metric/internal/jsonstrict"
)

type PrivacyRequest struct {
	SchemaVersion int    `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Action        string `json:"action"`
	RequestedAt   string `json:"requested_at"`
}

type InternalPrivacyRequest struct {
	PrivacyRequest
	InstallationID string `json:"installation_id"`
}

type ValidatedPrivacyRequest struct {
	InternalPrivacyRequest
	RequestedAtTime time.Time
}

func ParsePublicPrivacyRequest(data []byte, installationID string, now time.Time) (ValidatedPrivacyRequest, error) {
	var request PrivacyRequest
	if err := jsonstrict.DecodeObject(data, &request, 8); err != nil {
		return ValidatedPrivacyRequest{}, err
	}
	return validatePrivacyRequest(InternalPrivacyRequest{PrivacyRequest: request, InstallationID: installationID}, now)
}

func ParseInternalPrivacyRequest(data []byte, now time.Time) (ValidatedPrivacyRequest, error) {
	var request InternalPrivacyRequest
	if err := jsonstrict.DecodeObject(data, &request, 8); err != nil {
		return ValidatedPrivacyRequest{}, err
	}
	return validatePrivacyRequest(request, now)
}

func validatePrivacyRequest(request InternalPrivacyRequest, now time.Time) (ValidatedPrivacyRequest, error) {
	if request.SchemaVersion != SchemaVersion || request.Action != "delete" {
		return ValidatedPrivacyRequest{}, errors.New("invalid V1 privacy request")
	}
	if err := validateUUIDs(request.RequestID, request.InstallationID); err != nil {
		return ValidatedPrivacyRequest{}, err
	}
	requestedAt, err := parseMillisecondTime(request.RequestedAt)
	if err != nil || requestedAt.Before(now.Add(-24*time.Hour)) || requestedAt.After(now.Add(5*time.Minute)) {
		return ValidatedPrivacyRequest{}, errors.New("requested_at is outside the allowed range")
	}
	return ValidatedPrivacyRequest{InternalPrivacyRequest: request, RequestedAtTime: requestedAt}, nil
}
