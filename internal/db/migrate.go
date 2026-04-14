package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed migrations/init.sql
var initSQL string

// Migrate applies schema from migrations/init.sql (idempotent CREATE / seed).
func Migrate(db *sql.DB) error {
	for _, stmt := range splitStatements(initSQL) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w\n---\n%s\n---", err, stmt)
		}
	}
	return UpgradeSchema(db)
}

func splitStatements(sql string) []string {
	var b strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	var out []string
	for _, part := range strings.Split(b.String(), ";") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
