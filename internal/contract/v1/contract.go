package v1

import (
	"encoding/json"
	"time"
)

const (
	SchemaVersion   = 1
	MaxBatchRecords = 500
	MaxBatchBytes   = 512 * 1024
)

type Batch struct {
	SchemaVersion    int              `json:"schema_version"`
	Events           []Event          `json:"events"`
	SessionSummaries []SessionSummary `json:"session_summaries,omitempty"`
}

type Event struct {
	EventID        string          `json:"event_id"`
	EventName      string          `json:"event_name"`
	OccurredAt     string          `json:"occurred_at"`
	InstallationID string          `json:"installation_id"`
	AppSessionID   string          `json:"app_session_id"`
	GameSessionID  *string         `json:"game_session_id,omitempty"`
	App            AppMetadata     `json:"app"`
	Properties     json.RawMessage `json:"properties"`
}

type AppMetadata struct {
	Version   string `json:"version"`
	Build     string `json:"build"`
	Platform  string `json:"platform"`
	OSVersion string `json:"os_version"`
	Locale    string `json:"locale"`
}

type SessionSummary struct {
	SessionID          string      `json:"session_id"`
	SessionType        string      `json:"session_type"`
	InstallationID     string      `json:"installation_id"`
	AppSessionID       string      `json:"app_session_id"`
	GameSessionID      *string     `json:"game_session_id,omitempty"`
	GameID             *string     `json:"game_id,omitempty"`
	StartedAt          string      `json:"started_at"`
	EndedAt            string      `json:"ended_at"`
	DurationMS         uint64      `json:"duration_ms"`
	PausedMS           uint64      `json:"paused_ms,omitempty"`
	MenuMode           string      `json:"menu_mode,omitempty"`
	GameCategory       string      `json:"game_category,omitempty"`
	InputMethod        string      `json:"input_method,omitempty"`
	FinishReason       string      `json:"finish_reason,omitempty"`
	StepsCompleted     uint32      `json:"steps_completed,omitempty"`
	MaxSteps           uint32      `json:"max_steps,omitempty"`
	SuccessCount       uint32      `json:"success_count"`
	MistakeCount       uint32      `json:"mistake_count"`
	HintCount          uint32      `json:"hint_count"`
	TargetCancelCount  uint32      `json:"target_cancel_count,omitempty"`
	GazeLostCount      uint32      `json:"gaze_lost_count,omitempty"`
	DifficultyChanges  uint32      `json:"difficulty_changes,omitempty"`
	GazeSampleCount    uint64      `json:"gaze_sample_count,omitempty"`
	MouseSampleCount   uint64      `json:"mouse_sample_count,omitempty"`
	ValidGazeRatio     *float64    `json:"valid_gaze_ratio,omitempty"`
	MeanDwellMS        *float64    `json:"mean_dwell_ms,omitempty"`
	ConfiguredDwellMS  uint32      `json:"configured_dwell_ms,omitempty"`
	Result             string      `json:"result,omitempty"`
	InterruptionReason string      `json:"interruption_reason,omitempty"`
	App                AppMetadata `json:"app"`
}

type Dimensions struct {
	Page             *string
	Mode             *string
	GameCategory     *string
	SettingKey       *string
	SettingEnabled   *bool
	SettingNumber    *float64
	GameID           *string
	LevelIndex       *uint32
	TargetKind       *string
	InputMethod      *string
	ElapsedMS        *uint32
	ResponseMS       *uint32
	Result           *string
	Reason           *string
	HintKind         *string
	Difficulty       *uint8
	TobiiState       *string
	UpdaterState     *string
	UpdaterVersion   *string
	ErrorFingerprint *string
	ErrorComponent   *string
	DroppedCount     *uint64
}

type ValidatedEvent struct {
	Event
	OccurredAtTime time.Time
	Dimensions     Dimensions
}

type ValidatedSummary struct {
	SessionSummary
	StartedAtTime time.Time
	EndedAtTime   time.Time
}

type ValidatedBatch struct {
	Events           []ValidatedEvent
	SessionSummaries []ValidatedSummary
}

type EmptyProperties struct{}

type PageProperties struct {
	Page string `json:"page"`
}

type ModeProperties struct {
	Mode string `json:"mode"`
}

type SettingsProperties struct {
	SettingKey string   `json:"setting_key"`
	Enabled    *bool    `json:"enabled,omitempty"`
	Number     *float64 `json:"number,omitempty"`
}

type GameProperties struct {
	GameID       string `json:"game_id"`
	Mode         string `json:"mode,omitempty"`
	GameCategory string `json:"game_category,omitempty"`
}

type GameFinishedProperties struct {
	GameID string `json:"game_id"`
	Result string `json:"result,omitempty"`
	Reason string `json:"reason"`
}

type GameInterruptedProperties struct {
	GameID string `json:"game_id"`
	Reason string `json:"reason"`
}

type LevelProperties struct {
	GameID     string  `json:"game_id"`
	LevelIndex *uint32 `json:"level_index"`
}

type TargetProperties struct {
	GameID      string  `json:"game_id"`
	LevelIndex  *uint32 `json:"level_index"`
	TargetKind  string  `json:"target_kind"`
	InputMethod string  `json:"input_method"`
	ElapsedMS   *uint32 `json:"elapsed_ms,omitempty"`
	Reason      string  `json:"reason,omitempty"`
}

type OutcomeProperties struct {
	GameID      string  `json:"game_id"`
	LevelIndex  *uint32 `json:"level_index"`
	TargetKind  string  `json:"target_kind,omitempty"`
	InputMethod string  `json:"input_method"`
	ResponseMS  *uint32 `json:"response_ms,omitempty"`
}

type HintProperties struct {
	GameID     string  `json:"game_id"`
	LevelIndex *uint32 `json:"level_index"`
	HintKind   string  `json:"hint_kind"`
}

type DifficultyProperties struct {
	GameID     string `json:"game_id"`
	Difficulty uint8  `json:"difficulty"`
}

type TobiiProperties struct {
	State string `json:"state"`
}

type UpdaterProperties struct {
	State   string `json:"state"`
	Version string `json:"version,omitempty"`
}

type ErrorProperties struct {
	Fingerprint string `json:"fingerprint"`
	Component   string `json:"component"`
}

type QueueDroppedProperties struct {
	DroppedCount uint64 `json:"dropped_count"`
	Reason       string `json:"reason"`
}
