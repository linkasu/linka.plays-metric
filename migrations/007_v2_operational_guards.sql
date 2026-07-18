ALTER TABLE linka_metric.privacy_suppressions_v2
    ADD COLUMN IF NOT EXISTS attempts UInt16 DEFAULT 0;

ALTER TABLE linka_metric.privacy_suppressions_v2
    ADD COLUMN IF NOT EXISTS available_at DateTime64(3, 'UTC') DEFAULT ingested_at;

ALTER TABLE linka_metric.privacy_suppressions_v2
    ADD COLUMN IF NOT EXISTS lease_until Nullable(DateTime64(3, 'UTC'));

CREATE TABLE IF NOT EXISTS linka_metric.privacy_deletion_progress_v2
(
    request_id UUID,
    product LowCardinality(String),
    table_name LowCardinality(String),
    status LowCardinality(String),
    attempts UInt16,
    available_at DateTime64(3, 'UTC'),
    lease_until Nullable(DateTime64(3, 'UTC')),
    last_error Nullable(String),
    updated_at DateTime64(3, 'UTC'),
    completed_at Nullable(DateTime64(3, 'UTC'))
)
ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (product, request_id, table_name);

CREATE TABLE IF NOT EXISTS linka_metric.record_registry_v2
(
    product LowCardinality(String),
    record_id UUID,
    batch_id UUID,
    stream LowCardinality(String),
    body_sha256 FixedString(64),
    registered_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(registered_at)
ORDER BY (product, record_id);
