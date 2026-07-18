ALTER TABLE linka_metric.privacy_suppressions_v2
    ADD COLUMN IF NOT EXISTS legacy_installation_id Nullable(UUID);
