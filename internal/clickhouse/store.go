package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	v1 "github.com/linkasu/linka.plays-metric/internal/contract/v1"
)

type Config struct {
	Addresses []string
	Database  string
	Username  string
	Password  string
	Secure    bool
}

type Store struct {
	connection ch.Conn
}

func Open(config Config) (*Store, error) {
	options := &ch.Options{
		Addr: config.Addresses,
		Auth: ch.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		DialTimeout:     5 * time.Second,
		ConnMaxLifetime: time.Hour,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		Compression:     &ch.Compression{Method: ch.CompressionLZ4},
		Settings:        ch.Settings{"async_insert": 0},
	}
	if config.Secure {
		options.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	connection, err := ch.Open(options)
	if err != nil {
		return nil, fmt.Errorf("open ClickHouse: %w", err)
	}
	return &Store{connection: connection}, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.connection.Ping(ctx)
}

func (s *Store) Close() error {
	return s.connection.Close()
}

func (s *Store) Insert(ctx context.Context, batch v1.ValidatedBatch) error {
	ingestedAt := time.Now().UTC()
	if len(batch.Events) > 0 {
		if err := s.insertEvents(ctx, batch.Events, ingestedAt); err != nil {
			return err
		}
	}
	if len(batch.SessionSummaries) > 0 {
		if err := s.insertSummaries(ctx, batch.SessionSummaries, ingestedAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) insertEvents(ctx context.Context, events []v1.ValidatedEvent, ingestedAt time.Time) error {
	batch, err := s.connection.PrepareBatch(ctx, `INSERT INTO events (
		event_id, event_name, occurred_at, installation_id, app_session_id, game_session_id,
		app_version, app_build, platform, os_version, locale, page, mode, game_category, setting_key,
		setting_enabled, setting_number, game_id, level_index, target_kind, input_method, elapsed_ms, response_ms, result, reason,
		hint_kind, difficulty, tobii_state, updater_state, updater_version, error_fingerprint,
		error_component, dropped_count, ingested_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare events batch: %w", err)
	}
	for _, event := range events {
		dimensions := event.Dimensions
		if err := batch.Append(
			uuid.MustParse(event.EventID),
			event.EventName,
			event.OccurredAtTime,
			uuid.MustParse(event.InstallationID),
			uuid.MustParse(event.AppSessionID),
			nullableUUID(event.GameSessionID),
			event.App.Version,
			event.App.Build,
			event.App.Platform,
			event.App.OSVersion,
			event.App.Locale,
			dimensions.Page,
			dimensions.Mode,
			dimensions.GameCategory,
			dimensions.SettingKey,
			dimensions.SettingEnabled,
			dimensions.SettingNumber,
			dimensions.GameID,
			dimensions.LevelIndex,
			dimensions.TargetKind,
			dimensions.InputMethod,
			dimensions.ElapsedMS,
			dimensions.ResponseMS,
			dimensions.Result,
			dimensions.Reason,
			dimensions.HintKind,
			dimensions.Difficulty,
			dimensions.TobiiState,
			dimensions.UpdaterState,
			dimensions.UpdaterVersion,
			dimensions.ErrorFingerprint,
			dimensions.ErrorComponent,
			dimensions.DroppedCount,
			ingestedAt,
		); err != nil {
			return fmt.Errorf("append event: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send events batch: %w", err)
	}
	return nil
}

func (s *Store) insertSummaries(ctx context.Context, summaries []v1.ValidatedSummary, ingestedAt time.Time) error {
	batch, err := s.connection.PrepareBatch(ctx, `INSERT INTO session_summaries (
		session_id, session_type, installation_id, app_session_id, game_session_id, game_id,
		started_at, ended_at, duration_ms, paused_ms, menu_mode, game_category, input_method,
		finish_reason, steps_completed, max_steps, success_count, mistake_count, hint_count,
		target_cancel_count, gaze_lost_count, difficulty_changes, gaze_sample_count, mouse_sample_count,
		valid_gaze_ratio, mean_dwell_ms, configured_dwell_ms, result,
		interruption_reason, app_version, app_build, platform, os_version, locale, ingested_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare session summaries batch: %w", err)
	}
	for _, summary := range summaries {
		if err := batch.Append(
			uuid.MustParse(summary.SessionID),
			summary.SessionType,
			uuid.MustParse(summary.InstallationID),
			uuid.MustParse(summary.AppSessionID),
			nullableUUID(summary.GameSessionID),
			summary.GameID,
			summary.StartedAtTime,
			summary.EndedAtTime,
			summary.DurationMS,
			summary.PausedMS,
			nullableString(summary.MenuMode),
			nullableString(summary.GameCategory),
			nullableString(summary.InputMethod),
			nullableString(summary.FinishReason),
			summary.StepsCompleted,
			summary.MaxSteps,
			summary.SuccessCount,
			summary.MistakeCount,
			summary.HintCount,
			summary.TargetCancelCount,
			summary.GazeLostCount,
			summary.DifficultyChanges,
			summary.GazeSampleCount,
			summary.MouseSampleCount,
			summary.ValidGazeRatio,
			summary.MeanDwellMS,
			summary.ConfiguredDwellMS,
			nullableString(summary.Result),
			nullableString(summary.InterruptionReason),
			summary.App.Version,
			summary.App.Build,
			summary.App.Platform,
			summary.App.OSVersion,
			summary.App.Locale,
			ingestedAt,
		); err != nil {
			return fmt.Errorf("append session summary: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send session summaries batch: %w", err)
	}
	return nil
}

func nullableUUID(value *string) *uuid.UUID {
	if value == nil {
		return nil
	}
	parsed := uuid.MustParse(*value)
	return &parsed
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
