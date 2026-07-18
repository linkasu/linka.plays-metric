CREATE TABLE IF NOT EXISTS linka_metric.schema_migrations
(
    version UInt32,
    name String,
    checksum FixedString(64),
    applied_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(applied_at)
ORDER BY version;
