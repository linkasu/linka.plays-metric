package migrations

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

var databasePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

type ClickHouseBackend struct {
	connection ch.Conn
	database   string
}

func NewClickHouseBackend(connection ch.Conn, database string) (*ClickHouseBackend, error) {
	if !databasePattern.MatchString(database) {
		return nil, errors.New("invalid ClickHouse database name")
	}
	return &ClickHouseBackend{connection: connection, database: database}, nil
}

func (b *ClickHouseBackend) Prepare(ctx context.Context) error {
	if err := b.connection.Exec(ctx, "CREATE DATABASE IF NOT EXISTS "+b.database); err != nil {
		return err
	}
	return b.connection.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+b.database+`.schema_migrations (
		version UInt32,
		name String,
		checksum FixedString(64),
		applied_at DateTime64(3, 'UTC')
	) ENGINE = ReplacingMergeTree(applied_at) ORDER BY version`)
}

func (b *ClickHouseBackend) Applied(ctx context.Context) ([]AppliedMigration, error) {
	rows, err := b.connection.Query(ctx, `SELECT version, name, checksum FROM `+b.database+`.schema_migrations FINAL ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []AppliedMigration
	for rows.Next() {
		var row AppliedMigration
		if err := rows.Scan(&row.Version, &row.Name, &row.Checksum); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (b *ClickHouseBackend) Execute(ctx context.Context, statement string) error {
	return b.connection.Exec(ctx, statement)
}

func (b *ClickHouseBackend) Record(ctx context.Context, migration Migration) error {
	err := b.connection.Exec(ctx, `INSERT INTO `+b.database+`.schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
		migration.Version, migration.Name, migration.Checksum, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert ledger row: %w", err)
	}
	return nil
}
