package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed *.sql
var files embed.FS

type Migration struct {
	Version    uint32
	Name       string
	Checksum   string
	Statements []string
}

type AppliedMigration struct {
	Version  uint32
	Name     string
	Checksum string
}

type Backend interface {
	Prepare(context.Context) error
	Applied(context.Context) ([]AppliedMigration, error)
	Execute(context.Context, string) error
	Record(context.Context, Migration) error
}

func Load() ([]Migration, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("list embedded migrations: %w", err)
	}
	sort.Strings(names)
	migrations := make([]Migration, 0, len(names))
	versions := make(map[uint32]struct{}, len(names))
	for _, name := range names {
		separator := strings.IndexByte(name, '_')
		if separator != 3 {
			return nil, fmt.Errorf("migration %q must start with a three-digit version", name)
		}
		version64, err := strconv.ParseUint(name[:separator], 10, 32)
		if err != nil || version64 == 0 {
			return nil, fmt.Errorf("migration %q has invalid version", name)
		}
		version := uint32(version64)
		if _, exists := versions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d", version)
		}
		versions[version] = struct{}{}
		contents, err := files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", name, err)
		}
		digest := sha256.Sum256(contents)
		statements, err := splitStatements(string(contents))
		if err != nil {
			return nil, fmt.Errorf("parse migration %q: %w", name, err)
		}
		if len(statements) == 0 {
			return nil, fmt.Errorf("migration %q is empty", name)
		}
		migrations = append(migrations, Migration{
			Version: version, Name: name, Checksum: hex.EncodeToString(digest[:]), Statements: statements,
		})
	}
	return migrations, nil
}

func Run(ctx context.Context, backend Backend) error {
	migrations, err := Load()
	if err != nil {
		return err
	}
	if err := backend.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare migration ledger: %w", err)
	}
	appliedRows, err := backend.Applied(ctx)
	if err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	applied := make(map[uint32]AppliedMigration, len(appliedRows))
	embedded := make(map[uint32]struct{}, len(migrations))
	for _, migration := range migrations {
		embedded[migration.Version] = struct{}{}
	}
	for _, row := range appliedRows {
		if _, ok := embedded[row.Version]; !ok {
			return fmt.Errorf("migration ledger contains unknown version %d; refusing binary rollback", row.Version)
		}
		if existing, ok := applied[row.Version]; ok && existing.Checksum != row.Checksum {
			return fmt.Errorf("migration ledger contains conflicting checksums for version %d", row.Version)
		}
		applied[row.Version] = row
	}
	for _, migration := range migrations {
		if row, ok := applied[migration.Version]; ok {
			if row.Name != migration.Name || row.Checksum != migration.Checksum {
				return fmt.Errorf("migration %03d checksum mismatch: applied %s, embedded %s", migration.Version, row.Checksum, migration.Checksum)
			}
			continue
		}
		for index, statement := range migration.Statements {
			if err := backend.Execute(ctx, statement); err != nil {
				return fmt.Errorf("apply migration %03d statement %d: %w", migration.Version, index+1, err)
			}
		}
		if err := backend.Record(ctx, migration); err != nil {
			return fmt.Errorf("record migration %03d: %w", migration.Version, err)
		}
	}
	return nil
}

func splitStatements(input string) ([]string, error) {
	var statements []string
	var current strings.Builder
	var quote rune
	lineComment := false
	blockComment := false
	escaped := false
	runes := []rune(input)
	flush := func() {
		statement := strings.TrimSpace(current.String())
		if statement != "" {
			statements = append(statements, statement)
		}
		current.Reset()
	}

	for index := 0; index < len(runes); index++ {
		char := runes[index]
		next := rune(0)
		if index+1 < len(runes) {
			next = runes[index+1]
		}
		if lineComment {
			if char == '\n' {
				lineComment = false
				current.WriteRune(char)
			}
			continue
		}
		if blockComment {
			if char == '*' && next == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if quote != 0 {
			current.WriteRune(char)
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				if next == quote {
					current.WriteRune(next)
					index++
					continue
				}
				quote = 0
			}
			continue
		}
		switch {
		case char == '-' && next == '-':
			current.WriteRune(' ')
			lineComment = true
			index++
		case char == '/' && next == '*':
			current.WriteRune(' ')
			blockComment = true
			index++
		case char == '\'' || char == '"' || char == '`':
			quote = char
			current.WriteRune(char)
		case char == ';':
			flush()
		default:
			current.WriteRune(char)
		}
	}
	if quote != 0 || blockComment {
		return nil, errors.New("unterminated SQL quote or block comment")
	}
	flush()
	return statements, nil
}
