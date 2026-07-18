package privacy

import (
	"context"
	"errors"
	"fmt"
	"time"

	v2 "github.com/linkasu/linka.plays-metric/internal/contract/v2"
)

type Request struct {
	RequestID            string
	Scope                v2.Scope
	ProductKey           string
	Action               v2.PrivacyAction
	BodySHA256           string
	RequestedAt          time.Time
	IngestedAt           time.Time
	ExpiresAt            *time.Time
	Attempts             uint16
	LegacyInstallationID *string
}

type Repository interface {
	PendingPrivacyRequests(context.Context, int) ([]Request, error)
	ClaimPrivacyRequest(context.Context, Request) (bool, error)
	CompletePrivacyRequest(context.Context, Request) error
	RetryPrivacyRequest(context.Context, Request, string, int) error
}

type Deleter interface {
	DeleteTelemetryRequest(context.Context, Request) error
}

type Worker struct {
	repository  Repository
	deleter     Deleter
	limit       int
	maxAttempts int
}

func NewWorker(repository Repository, deleter Deleter, limit int) (*Worker, error) {
	return NewWorkerWithMaxAttempts(repository, deleter, limit, 10)
}

func NewWorkerWithMaxAttempts(repository Repository, deleter Deleter, limit, maxAttempts int) (*Worker, error) {
	if repository == nil || deleter == nil || limit < 1 || limit > 1000 || maxAttempts < 1 || maxAttempts > 100 {
		return nil, errors.New("invalid privacy worker configuration")
	}
	return &Worker{repository: repository, deleter: deleter, limit: limit, maxAttempts: maxAttempts}, nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	requests, err := w.repository.PendingPrivacyRequests(ctx, w.limit)
	if err != nil {
		return fmt.Errorf("list privacy requests: %w", err)
	}
	var failures []error
	for _, request := range requests {
		claimed, err := w.repository.ClaimPrivacyRequest(ctx, request)
		if err != nil {
			failures = append(failures, fmt.Errorf("claim privacy request %s: %w", request.RequestID, err))
			continue
		}
		if !claimed {
			continue
		}
		request.Attempts++
		switch request.Action {
		case v2.PrivacyOptOut:
			err = nil // The API persisted the active suppression before returning 202.
		case v2.PrivacyDelete:
			err = w.deleter.DeleteTelemetryRequest(ctx, request)
		default:
			err = errors.New("unsupported privacy action")
		}
		if err != nil {
			if markErr := w.repository.RetryPrivacyRequest(ctx, request, "processing_failed", w.maxAttempts); markErr != nil {
				failures = append(failures, fmt.Errorf("privacy request %s failed: %v; persist failure: %w", request.RequestID, err, markErr))
			} else {
				failures = append(failures, fmt.Errorf("privacy request %s failed: %w", request.RequestID, err))
			}
			continue
		}
		if err := w.repository.CompletePrivacyRequest(ctx, request); err != nil {
			failures = append(failures, fmt.Errorf("complete privacy request %s: %w", request.RequestID, err))
		}
	}
	return errors.Join(failures...)
}
