package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var safeValuePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:+-]{0,95}$`)
var fingerprintPattern = regexp.MustCompile(`^(?:[A-Fa-f0-9]{16,96}|sha256:[A-Fa-f0-9]{64})$`)

var knownEventNames = map[string]struct{}{
	"installation_created":     {},
	"app_started":              {},
	"app_backgrounded":         {},
	"app_foregrounded":         {},
	"app_closed":               {},
	"page_viewed":              {},
	"mode_changed":             {},
	"settings_changed":         {},
	"game_session_started":     {},
	"game_session_paused":      {},
	"game_session_resumed":     {},
	"game_session_finished":    {},
	"game_session_interrupted": {},
	"level_entered":            {},
	"level_cancelled":          {},
	"level_clicked":            {},
	"target_entered":           {},
	"target_cancelled":         {},
	"target_clicked":           {},
	"success":                  {},
	"mistake":                  {},
	"hint_used":                {},
	"difficulty_changed":       {},
	"tobii_state_changed":      {},
	"updater_state_changed":    {},
	"error":                    {},
	"queue_dropped":            {},
}

func ParseBatch(data []byte) (ValidatedBatch, error) {
	var batch Batch
	if err := decodeStrict(data, &batch); err != nil {
		return ValidatedBatch{}, fmt.Errorf("decode batch: %w", err)
	}
	if batch.SchemaVersion != SchemaVersion {
		return ValidatedBatch{}, fmt.Errorf("unsupported schema_version %d", batch.SchemaVersion)
	}
	recordCount := len(batch.Events) + len(batch.SessionSummaries)
	if recordCount == 0 || recordCount > MaxBatchRecords {
		return ValidatedBatch{}, fmt.Errorf("batch must contain between 1 and %d records", MaxBatchRecords)
	}
	validated := ValidatedBatch{
		Events:           make([]ValidatedEvent, 0, len(batch.Events)),
		SessionSummaries: make([]ValidatedSummary, 0, len(batch.SessionSummaries)),
	}
	for i := range batch.Events {
		event, err := validateEvent(batch.Events[i])
		if err != nil {
			return ValidatedBatch{}, fmt.Errorf("events[%d]: %w", i, err)
		}
		validated.Events = append(validated.Events, event)
	}
	for i := range batch.SessionSummaries {
		summary, err := validateSummary(batch.SessionSummaries[i])
		if err != nil {
			return ValidatedBatch{}, fmt.Errorf("session_summaries[%d]: %w", i, err)
		}
		validated.SessionSummaries = append(validated.SessionSummaries, summary)
	}
	return validated, nil
}

func validateEvent(event Event) (ValidatedEvent, error) {
	if err := validateUUIDs(event.EventID, event.InstallationID, event.AppSessionID); err != nil {
		return ValidatedEvent{}, err
	}
	if event.GameSessionID != nil {
		if err := validateUUIDs(*event.GameSessionID); err != nil {
			return ValidatedEvent{}, fmt.Errorf("game_session_id: %w", err)
		}
	}
	if _, ok := knownEventNames[event.EventName]; !ok {
		return ValidatedEvent{}, errors.New("unknown event_name")
	}
	if err := validateApp(event.App); err != nil {
		return ValidatedEvent{}, err
	}
	occurredAt, err := parseMillisecondTime(event.OccurredAt)
	if err != nil {
		return ValidatedEvent{}, fmt.Errorf("occurred_at: %w", err)
	}
	dimensions, gameEvent, err := validateProperties(event.EventName, event.Properties)
	if err != nil {
		return ValidatedEvent{}, fmt.Errorf("properties: %w", err)
	}
	if gameEvent && event.GameSessionID == nil {
		return ValidatedEvent{}, errors.New("game_session_id is required for game events")
	}
	return ValidatedEvent{Event: event, OccurredAtTime: occurredAt, Dimensions: dimensions}, nil
}

func validateProperties(name string, raw json.RawMessage) (Dimensions, bool, error) {
	var dimensions Dimensions
	gameEvent := false
	switch name {
	case "installation_created", "app_started", "app_backgrounded", "app_foregrounded", "app_closed":
		var properties EmptyProperties
		return dimensions, false, decodeStrict(raw, &properties)
	case "page_viewed":
		var properties PageProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("page", properties.Page); err != nil {
			return dimensions, false, err
		}
		dimensions.Page = &properties.Page
	case "mode_changed":
		var properties ModeProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("mode", properties.Mode); err != nil {
			return dimensions, false, err
		}
		dimensions.Mode = &properties.Mode
	case "settings_changed":
		var properties SettingsProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("setting_key", properties.SettingKey); err != nil {
			return dimensions, false, err
		}
		if (properties.Enabled == nil) == (properties.Number == nil) {
			return dimensions, false, errors.New("exactly one of enabled or number is required")
		}
		if properties.Number != nil && (*properties.Number < 0 || *properties.Number > 100000) {
			return dimensions, false, errors.New("number is outside the safe range")
		}
		dimensions.SettingKey = &properties.SettingKey
		dimensions.SettingEnabled = properties.Enabled
		dimensions.SettingNumber = properties.Number
	case "game_session_started", "game_session_paused", "game_session_resumed":
		var properties GameProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("game_id", properties.GameID); err != nil {
			return dimensions, false, err
		}
		dimensions.GameID = &properties.GameID
		if err := validateOptionalSafe("mode", properties.Mode); err != nil {
			return dimensions, false, err
		}
		if err := validateOptionalSafe("game_category", properties.GameCategory); err != nil {
			return dimensions, false, err
		}
		dimensions.Mode = optionalString(properties.Mode)
		dimensions.GameCategory = optionalString(properties.GameCategory)
		gameEvent = true
	case "game_session_finished":
		var properties GameFinishedProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("game_id", properties.GameID); err != nil {
			return dimensions, false, err
		}
		if err := validateOptionalSafe("result", properties.Result); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("reason", properties.Reason); err != nil {
			return dimensions, false, err
		}
		dimensions.GameID = &properties.GameID
		dimensions.Result = optionalString(properties.Result)
		dimensions.Reason = &properties.Reason
		gameEvent = true
	case "game_session_interrupted":
		var properties GameInterruptedProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("game_id", properties.GameID); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("reason", properties.Reason); err != nil {
			return dimensions, false, err
		}
		dimensions.GameID = &properties.GameID
		dimensions.Reason = &properties.Reason
		gameEvent = true
	case "level_entered", "level_cancelled", "level_clicked":
		var properties LevelProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("game_id", properties.GameID); err != nil {
			return dimensions, false, err
		}
		if properties.LevelIndex == nil {
			return dimensions, false, errors.New("level_index is required")
		}
		dimensions.GameID = &properties.GameID
		dimensions.LevelIndex = properties.LevelIndex
		gameEvent = true
	case "target_entered", "target_cancelled", "target_clicked":
		var properties TargetProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateTarget(properties.GameID, properties.TargetKind, properties.InputMethod, true); err != nil {
			return dimensions, false, err
		}
		if properties.LevelIndex == nil {
			return dimensions, false, errors.New("level_index is required")
		}
		dimensions.GameID = &properties.GameID
		dimensions.LevelIndex = properties.LevelIndex
		dimensions.TargetKind = &properties.TargetKind
		dimensions.InputMethod = &properties.InputMethod
		if properties.ElapsedMS != nil && *properties.ElapsedMS > 60000 {
			return dimensions, false, errors.New("elapsed_ms exceeds one minute")
		}
		if err := validateOptionalSafe("reason", properties.Reason); err != nil {
			return dimensions, false, err
		}
		dimensions.ElapsedMS = properties.ElapsedMS
		dimensions.Reason = optionalString(properties.Reason)
		gameEvent = true
	case "success", "mistake":
		var properties OutcomeProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateTarget(properties.GameID, properties.TargetKind, properties.InputMethod, false); err != nil {
			return dimensions, false, err
		}
		if properties.LevelIndex == nil {
			return dimensions, false, errors.New("level_index is required")
		}
		dimensions.GameID = &properties.GameID
		dimensions.LevelIndex = properties.LevelIndex
		dimensions.TargetKind = optionalString(properties.TargetKind)
		dimensions.InputMethod = &properties.InputMethod
		if properties.ResponseMS != nil && *properties.ResponseMS > 24*60*60*1000 {
			return dimensions, false, errors.New("response_ms exceeds one day")
		}
		dimensions.ResponseMS = properties.ResponseMS
		gameEvent = true
	case "hint_used":
		var properties HintProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafeValues(map[string]string{"game_id": properties.GameID, "hint_kind": properties.HintKind}); err != nil {
			return dimensions, false, err
		}
		if properties.LevelIndex == nil {
			return dimensions, false, errors.New("level_index is required")
		}
		dimensions.GameID = &properties.GameID
		dimensions.LevelIndex = properties.LevelIndex
		dimensions.HintKind = &properties.HintKind
		gameEvent = true
	case "difficulty_changed":
		var properties DifficultyProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if err := validateSafe("game_id", properties.GameID); err != nil {
			return dimensions, false, err
		}
		if properties.Difficulty < 1 || properties.Difficulty > 10 {
			return dimensions, false, errors.New("difficulty must be between 1 and 10")
		}
		dimensions.GameID = &properties.GameID
		dimensions.Difficulty = &properties.Difficulty
		gameEvent = true
	case "tobii_state_changed":
		var properties TobiiProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if !oneOf(properties.State, "unsupported", "service_starting", "service_unavailable", "connecting", "waiting_device", "connected", "tracking", "reconnecting", "error") {
			return dimensions, false, errors.New("unknown Tobii state")
		}
		dimensions.TobiiState = &properties.State
	case "updater_state_changed":
		var properties UpdaterProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if !oneOf(properties.State, "idle", "checking", "available", "downloading", "downloaded", "installing", "error") {
			return dimensions, false, errors.New("unknown updater state")
		}
		if err := validateOptionalSafe("version", properties.Version); err != nil {
			return dimensions, false, err
		}
		dimensions.UpdaterState = &properties.State
		dimensions.UpdaterVersion = optionalString(properties.Version)
	case "error":
		var properties ErrorProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if !fingerprintPattern.MatchString(properties.Fingerprint) {
			return dimensions, false, errors.New("fingerprint must be a stable hexadecimal hash")
		}
		if err := validateSafe("component", properties.Component); err != nil {
			return dimensions, false, err
		}
		dimensions.ErrorFingerprint = &properties.Fingerprint
		dimensions.ErrorComponent = &properties.Component
	case "queue_dropped":
		var properties QueueDroppedProperties
		if err := decodeStrict(raw, &properties); err != nil {
			return dimensions, false, err
		}
		if properties.DroppedCount == 0 {
			return dimensions, false, errors.New("dropped_count must be positive")
		}
		if !oneOf(properties.Reason, "capacity", "expired", "invalid", "shutdown") {
			return dimensions, false, errors.New("unknown queue drop reason")
		}
		dimensions.DroppedCount = &properties.DroppedCount
		dimensions.Reason = &properties.Reason
	default:
		return dimensions, false, errors.New("unknown event_name")
	}
	return dimensions, gameEvent, nil
}

func validateSummary(summary SessionSummary) (ValidatedSummary, error) {
	if err := validateUUIDs(summary.SessionID, summary.InstallationID, summary.AppSessionID); err != nil {
		return ValidatedSummary{}, err
	}
	if err := validateApp(summary.App); err != nil {
		return ValidatedSummary{}, err
	}
	startedAt, err := parseMillisecondTime(summary.StartedAt)
	if err != nil {
		return ValidatedSummary{}, fmt.Errorf("started_at: %w", err)
	}
	endedAt, err := parseMillisecondTime(summary.EndedAt)
	if err != nil {
		return ValidatedSummary{}, fmt.Errorf("ended_at: %w", err)
	}
	if endedAt.Before(startedAt) {
		return ValidatedSummary{}, errors.New("ended_at precedes started_at")
	}
	if summary.DurationMS > uint64((7 * 24 * time.Hour).Milliseconds()) {
		return ValidatedSummary{}, errors.New("duration_ms exceeds seven days")
	}
	if err := validateOptionalSafe("result", summary.Result); err != nil {
		return ValidatedSummary{}, err
	}
	if err := validateOptionalSafe("interruption_reason", summary.InterruptionReason); err != nil {
		return ValidatedSummary{}, err
	}
	for name, value := range map[string]string{
		"menu_mode":     summary.MenuMode,
		"game_category": summary.GameCategory,
		"input_method":  summary.InputMethod,
		"finish_reason": summary.FinishReason,
	} {
		if err := validateOptionalSafe(name, value); err != nil {
			return ValidatedSummary{}, err
		}
	}
	if summary.ValidGazeRatio != nil && (*summary.ValidGazeRatio < 0 || *summary.ValidGazeRatio > 1) {
		return ValidatedSummary{}, errors.New("valid_gaze_ratio must be between zero and one")
	}
	if summary.MeanDwellMS != nil && (*summary.MeanDwellMS < 0 || *summary.MeanDwellMS > 60000) {
		return ValidatedSummary{}, errors.New("mean_dwell_ms is outside the safe range")
	}
	if summary.ConfiguredDwellMS > 60000 {
		return ValidatedSummary{}, errors.New("configured_dwell_ms exceeds one minute")
	}
	switch summary.SessionType {
	case "app":
		if summary.SessionID != summary.AppSessionID || summary.GameSessionID != nil || summary.GameID != nil {
			return ValidatedSummary{}, errors.New("invalid app session identifiers")
		}
	case "game":
		if summary.GameSessionID == nil || summary.GameID == nil || summary.SessionID != *summary.GameSessionID {
			return ValidatedSummary{}, errors.New("invalid game session identifiers")
		}
		if err := validateUUIDs(*summary.GameSessionID); err != nil {
			return ValidatedSummary{}, fmt.Errorf("game_session_id: %w", err)
		}
		if err := validateSafe("game_id", *summary.GameID); err != nil {
			return ValidatedSummary{}, err
		}
	default:
		return ValidatedSummary{}, errors.New("session_type must be app or game")
	}
	return ValidatedSummary{SessionSummary: summary, StartedAtTime: startedAt, EndedAtTime: endedAt}, nil
}

func validateApp(app AppMetadata) error {
	if err := validateSafeValues(map[string]string{
		"app.version":    app.Version,
		"app.build":      app.Build,
		"app.os_version": app.OSVersion,
		"app.locale":     app.Locale,
	}); err != nil {
		return err
	}
	if !oneOf(app.Platform, "windows", "macos", "linux") {
		return errors.New("app.platform must be windows, macos, or linux")
	}
	return nil
}

func validateTarget(gameID, targetKind, inputMethod string, targetRequired bool) error {
	if err := validateSafe("game_id", gameID); err != nil {
		return err
	}
	if targetRequired {
		if err := validateSafe("target_kind", targetKind); err != nil {
			return err
		}
	} else if err := validateOptionalSafe("target_kind", targetKind); err != nil {
		return err
	}
	if !oneOf(inputMethod, "mouse", "touch", "gaze", "keyboard") {
		return errors.New("unknown input_method")
	}
	return nil
}

func validateUUIDs(values ...string) error {
	for _, value := range values {
		if _, err := uuid.Parse(value); err != nil {
			return errors.New("invalid UUID")
		}
	}
	return nil
}

func parseMillisecondTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, errors.New("must be RFC3339")
	}
	if parsed.Nanosecond()%int(time.Millisecond) != 0 {
		return time.Time{}, errors.New("precision must not exceed milliseconds")
	}
	return parsed.UTC(), nil
}

func validateSafeValues(values map[string]string) error {
	for name, value := range values {
		if err := validateSafe(name, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSafe(name, value string) error {
	if !safeValuePattern.MatchString(value) {
		return fmt.Errorf("%s contains an unsafe or empty value", name)
	}
	return nil
}

func validateOptionalSafe(name, value string) error {
	if value == "" {
		return nil
	}
	return validateSafe(name, value)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func decodeStrict(data []byte, destination any) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("expected JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
