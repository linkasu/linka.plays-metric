CREATE TABLE IF NOT EXISTS linka_metric.ingest_batches_v2
(
    batch_id UUID,
    idempotency_key UUID,
    product LowCardinality(String),
    product_key Nullable(FixedString(64)),
    subject_key FixedString(64),
    person_key Nullable(FixedString(64)),
    org_key Nullable(FixedString(64)),
    stream LowCardinality(String),
    body_sha256 FixedString(64),
    record_count UInt16,
    sent_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC'),
    expires_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (product, batch_id);

CREATE TABLE IF NOT EXISTS linka_metric.privacy_suppressions_v2
(
    request_id UUID,
    product LowCardinality(String),
    product_key Nullable(FixedString(64)),
    subject_key FixedString(64),
    person_key Nullable(FixedString(64)),
    org_key Nullable(FixedString(64)),
    action LowCardinality(String),
    status LowCardinality(String),
    body_sha256 FixedString(64),
    requested_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC'),
    updated_at DateTime64(3, 'UTC'),
    active Bool,
    failure_code Nullable(String),
    expires_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (product, request_id);

CREATE TABLE IF NOT EXISTS linka_metric.common_events_v2
(
    product LowCardinality(String),
    product_key Nullable(FixedString(64)),
    subject_key FixedString(64),
    person_key Nullable(FixedString(64)),
    org_key Nullable(FixedString(64)),
    batch_id UUID,
    record_id UUID,
    occurred_at DateTime64(3, 'UTC'),
    kind LowCardinality(String),
    app_session_id UUID,
    app_version String,
    app_build String,
    platform LowCardinality(String),
    os_version String,
    locale LowCardinality(String),
    page Nullable(String),
    mode Nullable(String),
    ingested_at DateTime64(3, 'UTC'),
    expires_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (product, subject_key, record_id);

CREATE TABLE IF NOT EXISTS linka_metric.technical_events_v2
(
    product LowCardinality(String),
    product_key Nullable(FixedString(64)),
    subject_key FixedString(64),
    person_key Nullable(FixedString(64)),
    org_key Nullable(FixedString(64)),
    batch_id UUID,
    record_id UUID,
    occurred_at DateTime64(3, 'UTC'),
    kind LowCardinality(String),
    app_session_id UUID,
    app_version String,
    app_build String,
    platform LowCardinality(String),
    os_version String,
    locale LowCardinality(String),
    component LowCardinality(String),
    state Nullable(String),
    error_fingerprint Nullable(String),
    dropped_count Nullable(UInt64),
    drop_reason Nullable(String),
    ingested_at DateTime64(3, 'UTC'),
    expires_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (product, subject_key, record_id);

CREATE TABLE IF NOT EXISTS linka_metric.plays_events_v2
(
    product LowCardinality(String),
    product_key Nullable(FixedString(64)),
    subject_key FixedString(64),
    person_key Nullable(FixedString(64)),
    org_key Nullable(FixedString(64)),
    batch_id UUID,
    record_id UUID,
    occurred_at DateTime64(3, 'UTC'),
    kind LowCardinality(String),
    app_session_id UUID,
    game_session_id UUID,
    app_version String,
    app_build String,
    platform LowCardinality(String),
    os_version String,
    locale LowCardinality(String),
    game_id LowCardinality(String),
    game_category LowCardinality(String),
    input_method LowCardinality(String),
    level_index Nullable(UInt32),
    outcome Nullable(String),
    duration_ms Nullable(UInt64),
    success_count Nullable(UInt32),
    mistake_count Nullable(UInt32),
    hint_count Nullable(UInt32),
    valid_gaze_ratio Nullable(Float64),
    ingested_at DateTime64(3, 'UTC'),
    expires_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (product, subject_key, record_id);
