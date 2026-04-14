package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// UpgradeSchema adds columns/tables for databases created before schema changes.
func UpgradeSchema(db *sql.DB) error {
	type col struct {
		table string
		name  string
		typ   string
	}
	for _, c := range []col{
		{"boards", "max_threads", "INTEGER NOT NULL DEFAULT 0"},
		{"boards", "nsfw", "INTEGER NOT NULL DEFAULT 0"},
		{"threads", "bumped_at", "DATETIME"},
		{"threads", "pinned", "INTEGER NOT NULL DEFAULT 0"},
		{"threads", "archived", "INTEGER NOT NULL DEFAULT 0"},
		{"posts", "sage", "INTEGER NOT NULL DEFAULT 0"},
		{"posts", "hidden", "INTEGER NOT NULL DEFAULT 0"},
		{"posts", "poster_name", "TEXT"},
		{"posts", "trip_hash", "TEXT"},
		{"posts", "file_hash", "TEXT"},
		{"posts", "mime", "TEXT"},
		{"posts", "file_size", "INTEGER NOT NULL DEFAULT 0"},
		{"posts", "thumb_path", "TEXT"},
	} {
		if err := addColumnIfMissing(db, c.table, c.name, c.typ); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`
		UPDATE threads SET bumped_at = created_at
		WHERE bumped_at IS NULL
	`); err != nil {
		return err
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS bans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			board_id INTEGER,
			reason TEXT,
			expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (board_id) REFERENCES boards(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bans_ip ON bans(ip)`,
		`CREATE TABLE IF NOT EXISTS mod_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			detail TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			key TEXT PRIMARY KEY,
			response_body TEXT,
			status_code INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS post_edits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			old_text TEXT,
			edited_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("upgrade: %w", err)
		}
	}

	if err := ensureFTS(db); err != nil {
		return err
	}

	// Helpful index for bump order (ignore error if duplicate name on old sqlite)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_threads_bumped ON threads(board_id, pinned DESC, bumped_at DESC, id DESC)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_posts_file_hash ON posts(file_hash)`)

	return nil
}

func addColumnIfMissing(db *sql.DB, table, colName, colType string) error {
	exists, err := columnExists(db, table, colName)
	if err != nil || exists {
		return err
	}
	q := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, quoteIdent(table), quoteIdent(colName), colType)
	_, err = db.Exec(q)
	return err
}

func quoteIdent(s string) string {
	if strings.ContainsAny(s, `"' ;`) {
		return s
	}
	return s
}

func columnExists(db *sql.DB, table, colName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, quoteIdent(table)))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, colName) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func ensureFTS(db *sql.DB) error {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='posts_fts'`).Scan(&n)
	if n == 0 {
		if _, err := db.Exec(`
			CREATE VIRTUAL TABLE posts_fts USING fts5(
				text,
				content='posts',
				content_rowid='id'
			)
		`); err != nil {
			return fmt.Errorf("fts5 create: %w (возможно, сборка SQLite без FTS5)", err)
		}
		if _, err := db.Exec(`
			INSERT INTO posts_fts(rowid, text)
			SELECT id, COALESCE(text, '') FROM posts WHERE COALESCE(hidden, 0) = 0
		`); err != nil {
			return err
		}
	}

	triggers := []string{
		`CREATE TRIGGER IF NOT EXISTS posts_fts_ai AFTER INSERT ON posts
			WHEN COALESCE(NEW.hidden, 0) = 0
			BEGIN
				INSERT INTO posts_fts(rowid, text) VALUES (NEW.id, COALESCE(NEW.text, ''));
			END`,
		`CREATE TRIGGER IF NOT EXISTS posts_fts_ad AFTER DELETE ON posts BEGIN
				INSERT INTO posts_fts(posts_fts, rowid) VALUES('delete', OLD.id);
			END`,
		`CREATE TRIGGER IF NOT EXISTS posts_fts_au AFTER UPDATE ON posts BEGIN
				INSERT INTO posts_fts(posts_fts, rowid) VALUES('delete', OLD.id);
				INSERT INTO posts_fts(rowid, text) SELECT NEW.id, COALESCE(NEW.text, '')
					WHERE COALESCE(NEW.hidden, 0) = 0;
			END`,
	}
	for _, t := range triggers {
		if _, err := db.Exec(t); err != nil {
			return fmt.Errorf("fts trigger: %w", err)
		}
	}
	return nil
}
