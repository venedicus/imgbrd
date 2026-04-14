package repository

import (
	"github.com/venedicus/imgbrd/internal/model"
)

func (r *Repository) AddModLog(action, detail string) error {
	_, err := r.db.Exec(`INSERT INTO mod_log (action, detail) VALUES (?, ?)`, action, nullStr(detail))
	return err
}

func (r *Repository) ListModLog(limit int) ([]model.ModLogEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.db.Query(`
		SELECT id, action, COALESCE(detail, ''), created_at
		FROM mod_log
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.ModLogEntry
	for rows.Next() {
		var e model.ModLogEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
