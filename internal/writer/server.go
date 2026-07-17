package writer

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/linkasu/linka.plays-metric/internal/auth"
	v1 "github.com/linkasu/linka.plays-metric/internal/contract/v1"
	"github.com/linkasu/linka.plays-metric/internal/httpx"
)

type Store interface {
	Ping(context.Context) error
	Insert(context.Context, v1.ValidatedBatch) error
}

type Server struct {
	store   Store
	secret  []byte
	logger  *slog.Logger
	maxSkew time.Duration
	now     func() time.Time
}

func NewServer(store Store, secret []byte, logger *slog.Logger, maxSkew time.Duration) http.Handler {
	server := &Server{
		store:   store,
		secret:  append([]byte(nil), secret...),
		logger:  logger,
		maxSkew: maxSkew,
		now:     time.Now,
	}
	router := chi.NewRouter()
	router.Use(httpx.SecurityHeaders)
	router.Use(httpx.RequestLog(logger))
	router.Get("/healthz", server.health)
	router.Post("/internal/v1/events", server.events)
	return router
}

func (s *Server) health(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		httpx.WriteError(response, http.StatusServiceUnavailable, "store_unavailable")
		return
	}
	httpx.WriteJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) events(response http.ResponseWriter, request *http.Request) {
	if !httpx.IsJSON(request) {
		httpx.WriteError(response, http.StatusUnsupportedMediaType, "content_type_must_be_application_json")
		return
	}
	body, err := httpx.ReadBody(response, request, v1.MaxBatchBytes)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			httpx.WriteError(response, http.StatusRequestEntityTooLarge, "body_too_large")
			return
		}
		httpx.WriteError(response, http.StatusBadRequest, "invalid_body")
		return
	}
	if err := auth.VerifyWriterRequest(
		s.secret,
		body,
		s.now(),
		s.maxSkew,
		request.Header.Get(auth.WriterTimestampHeader),
		request.Header.Get(auth.WriterBodySHAHeader),
		request.Header.Get(auth.WriterSignatureHeader),
	); err != nil {
		httpx.WriteError(response, http.StatusUnauthorized, "invalid_writer_signature")
		return
	}
	batch, err := v1.ParseBatch(body)
	if err != nil {
		httpx.WriteError(response, http.StatusBadRequest, "invalid_event_batch")
		return
	}
	if err := s.store.Insert(request.Context(), batch); err != nil {
		s.logger.Error("insert event batch", "error", err)
		httpx.WriteError(response, http.StatusServiceUnavailable, "store_unavailable")
		return
	}
	httpx.WriteJSON(response, http.StatusAccepted, map[string]int{
		"inserted_events":            len(batch.Events),
		"inserted_session_summaries": len(batch.SessionSummaries),
	})
}
