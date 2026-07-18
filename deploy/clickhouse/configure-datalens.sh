#!/bin/sh
set -eu

env_file="${1:-.env}"
tmp_file="${env_file}.datalens.$$"
trap 'rm -f "$tmp_file"' EXIT
umask 077

IFS= read -r datalens_password
if [ -z "$datalens_password" ]; then
  printf '%s\n' "DataLens password is empty" >&2
  exit 1
fi

datalens_hash="$(printf '%s' "$datalens_password" | sha256sum | cut -d ' ' -f 1)"
password_written=false
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    CLICKHOUSE_DATALENS_PASSWORD=*)
      printf 'CLICKHOUSE_DATALENS_PASSWORD=%s\n' "$datalens_password" >> "$tmp_file"
      password_written=true
      ;;
    *)
      printf '%s\n' "$line" >> "$tmp_file"
      ;;
  esac
done < "$env_file"

if [ "$password_written" = false ]; then
  printf 'CLICKHOUSE_DATALENS_PASSWORD=%s\n' "$datalens_password" >> "$tmp_file"
fi
chmod 600 "$tmp_file"
mv "$tmp_file" "$env_file"
trap - EXIT
unset datalens_password

printf "ALTER USER datalens IDENTIFIED WITH sha256_hash BY '%s' SETTINGS readonly = 2;\n" "$datalens_hash" |
  docker compose exec -T clickhouse sh -c \
    'clickhouse-client --user "$CLICKHOUSE_ADMIN_USER" --password "$CLICKHOUSE_ADMIN_PASSWORD" --multiquery'
