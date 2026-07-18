ALTER TABLE linka_metric.record_registry_v2
    ADD COLUMN IF NOT EXISTS product_key Nullable(FixedString(64));
ALTER TABLE linka_metric.record_registry_v2
    ADD COLUMN IF NOT EXISTS subject_key Nullable(FixedString(64));
ALTER TABLE linka_metric.record_registry_v2
    ADD COLUMN IF NOT EXISTS person_key Nullable(FixedString(64));
ALTER TABLE linka_metric.record_registry_v2
    ADD COLUMN IF NOT EXISTS org_key Nullable(FixedString(64));
