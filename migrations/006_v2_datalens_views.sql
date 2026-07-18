CREATE VIEW IF NOT EXISTS linka_metric.datalens_common_v2
SQL SECURITY DEFINER
AS
SELECT
    product,
    if(product_key IS NULL, NULL, cityHash64(assumeNotNull(product_key))) AS product_key,
    cityHash64(subject_key) AS subject_key,
    if(person_key IS NULL, NULL, cityHash64(assumeNotNull(person_key))) AS person_key,
    if(org_key IS NULL, NULL, cityHash64(assumeNotNull(org_key))) AS org_key,
    record_id,
    occurred_at,
    kind,
    cityHash64(app_session_id) AS app_session_key,
    app_version,
    app_build,
    platform,
    os_version,
    locale,
    page,
    mode
FROM linka_metric.common_events_v2 AS events FINAL
WHERE (toString(ifNull(events.product_key, '')), toString(events.subject_key)) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(subject_key)
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.person_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(person_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND person_key IS NOT NULL
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.org_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(org_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND org_key IS NOT NULL
)
SETTINGS join_use_nulls = 1;

CREATE VIEW IF NOT EXISTS linka_metric.datalens_technical_v2
SQL SECURITY DEFINER
AS
SELECT
    product,
    if(product_key IS NULL, NULL, cityHash64(assumeNotNull(product_key))) AS product_key,
    cityHash64(subject_key) AS subject_key,
    if(person_key IS NULL, NULL, cityHash64(assumeNotNull(person_key))) AS person_key,
    if(org_key IS NULL, NULL, cityHash64(assumeNotNull(org_key))) AS org_key,
    record_id,
    occurred_at,
    kind,
    cityHash64(app_session_id) AS app_session_key,
    app_version,
    app_build,
    platform,
    os_version,
    locale,
    component,
    state,
    error_fingerprint,
    dropped_count,
    drop_reason
FROM linka_metric.technical_events_v2 AS events FINAL
WHERE (toString(ifNull(events.product_key, '')), toString(events.subject_key)) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(subject_key)
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.person_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(person_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND person_key IS NOT NULL
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.org_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(org_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND org_key IS NOT NULL
)
SETTINGS join_use_nulls = 1;

CREATE VIEW IF NOT EXISTS linka_metric.datalens_plays_v2
SQL SECURITY DEFINER
AS
SELECT
    product,
    if(product_key IS NULL, NULL, cityHash64(assumeNotNull(product_key))) AS product_key,
    cityHash64(subject_key) AS subject_key,
    if(person_key IS NULL, NULL, cityHash64(assumeNotNull(person_key))) AS person_key,
    if(org_key IS NULL, NULL, cityHash64(assumeNotNull(org_key))) AS org_key,
    record_id,
    occurred_at,
    kind,
    cityHash64(app_session_id) AS app_session_key,
    cityHash64(game_session_id) AS game_session_key,
    app_version,
    app_build,
    platform,
    os_version,
    locale,
    game_id,
    game_category,
    input_method,
    level_index,
    outcome,
    duration_ms,
    success_count,
    mistake_count,
    hint_count,
    valid_gaze_ratio
FROM linka_metric.plays_events_v2 AS events FINAL
WHERE (toString(ifNull(events.product_key, '')), toString(events.subject_key)) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(subject_key)
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.person_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(person_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND person_key IS NOT NULL
)
AND (toString(ifNull(events.product_key, '')), toString(ifNull(events.org_key, ''))) NOT IN
(
    SELECT toString(ifNull(product_key, '')), toString(assumeNotNull(org_key))
    FROM linka_metric.privacy_suppressions_v2 FINAL
    WHERE active = true AND org_key IS NOT NULL
)
SETTINGS join_use_nulls = 1;
