ALTER TABLE linka_metric.ingest_batches_v2
    ADD COLUMN IF NOT EXISTS status LowCardinality(String) DEFAULT 'completed';
