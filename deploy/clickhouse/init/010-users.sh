#!/bin/sh
set -eu

: "${CLICKHOUSE_ADMIN_USER:?CLICKHOUSE_ADMIN_USER is required}"
: "${CLICKHOUSE_ADMIN_PASSWORD:?CLICKHOUSE_ADMIN_PASSWORD is required}"
: "${CLICKHOUSE_WRITER_PASSWORD:?CLICKHOUSE_WRITER_PASSWORD is required}"
: "${CLICKHOUSE_DATALENS_PASSWORD:?CLICKHOUSE_DATALENS_PASSWORD is required}"

writer_hash="$(printf '%s' "$CLICKHOUSE_WRITER_PASSWORD" | sha256sum | cut -d ' ' -f 1)"
datalens_hash="$(printf '%s' "$CLICKHOUSE_DATALENS_PASSWORD" | sha256sum | cut -d ' ' -f 1)"

clickhouse-client \
  --user "$CLICKHOUSE_ADMIN_USER" \
  --password "$CLICKHOUSE_ADMIN_PASSWORD" \
  --multiquery \
  --query "
    CREATE USER OR REPLACE metric_writer
      IDENTIFIED WITH sha256_hash BY '${writer_hash}'
      HOST ANY;
    CREATE USER OR REPLACE datalens
      IDENTIFIED WITH sha256_hash BY '${datalens_hash}'
      HOST ANY
      SETTINGS readonly = 1;
    GRANT INSERT ON linka_metric.events TO metric_writer;
    GRANT INSERT ON linka_metric.session_summaries TO metric_writer;
    GRANT SELECT ON linka_metric.datalens_events TO datalens;
    GRANT SELECT ON linka_metric.datalens_session_summaries TO datalens;
  "
