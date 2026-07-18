package privacy

import (
	"context"
	"errors"
	"testing"

	v2 "github.com/linkasu/linka.plays-metric/internal/contract/v2"
)

type fakeRepository struct {
	requests  []Request
	completed int
	retried   int
}

func (r *fakeRepository) PendingPrivacyRequests(context.Context, int) ([]Request, error) {
	return r.requests, nil
}
func (r *fakeRepository) ClaimPrivacyRequest(context.Context, Request) (bool, error) {
	return true, nil
}
func (r *fakeRepository) CompletePrivacyRequest(context.Context, Request) error {
	r.completed++
	return nil
}
func (r *fakeRepository) RetryPrivacyRequest(context.Context, Request, string, int) error {
	r.retried++
	return nil
}

type fakeDeleter struct {
	err error
}

func (d fakeDeleter) DeleteTelemetryRequest(context.Context, Request) error { return d.err }

func TestWorkerDoesNotReportDeletionSuccessOnFailure(t *testing.T) {
	repository := &fakeRepository{requests: []Request{{RequestID: "request", Action: v2.PrivacyDelete}}}
	worker, err := NewWorker(repository, fakeDeleter{err: errors.New("mutation failed")}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("worker hid deletion failure")
	}
	if repository.completed != 0 || repository.retried != 1 {
		t.Fatalf("completed = %d, retried = %d", repository.completed, repository.retried)
	}
}

func TestWorkerCompletesPersistedOptOutWithoutDeletion(t *testing.T) {
	repository := &fakeRepository{requests: []Request{{RequestID: "request", Action: v2.PrivacyOptOut}}}
	worker, err := NewWorker(repository, fakeDeleter{err: errors.New("must not be called")}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.completed != 1 || repository.retried != 0 {
		t.Fatalf("completed = %d, retried = %d", repository.completed, repository.retried)
	}
}
