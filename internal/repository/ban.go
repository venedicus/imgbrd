package repository

import (
	"database/sql"
	"time"

	"github.com/venedicus/imgbrd/internal/model"
)

func (r *Repository) IsBanned(ip string, boardID int) (bool, error) {
	var n int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM bans
		WHERE ip = ?
			AND (board_id IS NULL OR board_id = ?)
			AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))
	`, ip, boardID).Scan(&n)
	return n > 0, err
}

func (r *Repository) AddBan(ip string, boardID *int, reason string, expiresAt *time.Time) error {
	var b interface{}
	if boardID != nil {
		b = *boardID
	}
	var exp interface{}
	if expiresAt != nil {
		exp = expiresAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.Exec(`
		INSERT INTO bans (ip, board_id, reason, expires_at)
		VALUES (?, ?, ?, ?)
	`, ip, b, nullStr(reason), exp)
	return err
}

func (r *Repository) ListBans(limit int) ([]model.Ban, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(`
		SELECT id, ip, board_id, reason, expires_at, created_at
		FROM bans
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Ban
	for rows.Next() {
		var b model.Ban
		var bid sql.NullInt64
		var exp sql.NullString
		if err := rows.Scan(&b.ID, &b.IP, &bid, &b.Reason, &exp, &b.CreatedAt); err != nil {
			return nil, err
		}
		if bid.Valid {
			v := int(bid.Int64)
			b.BoardID = &v
		}
		if exp.Valid && exp.String != "" {
			t, err := time.Parse(time.RFC3339, exp.String)
			if err == nil {
				b.ExpiresAt = &t
			}
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
