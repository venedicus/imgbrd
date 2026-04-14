package repository

import (
	"database/sql"
	"time"

	"github.com/venedicus/imgbrd/internal/model"
)

func effectiveBumpedAt(bumpedAt sql.NullTime, createdAt time.Time) time.Time {
	if bumpedAt.Valid {
		return bumpedAt.Time
	}
	return createdAt
}

func (r *Repository) GetThreadsByBoardID(boardID int) ([]model.Thread, error) {
	rows, err := r.db.Query(`
		SELECT
			t.id,
			t.board_id,
			t.title,
			t.bumped_at,
			COALESCE(t.pinned, 0),
			COALESCE(t.archived, 0),
			t.created_at,
			(
				SELECT COUNT(*)
				FROM threads t2
				WHERE t2.board_id = t.board_id
					AND t2.id <= t.id
			) AS board_thread_no
		FROM threads t
		WHERE t.board_id = ?
			AND COALESCE(t.archived, 0) = 0
		ORDER BY COALESCE(t.pinned, 0) DESC,
			datetime(COALESCE(t.bumped_at, t.created_at)) DESC,
			t.id DESC
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []model.Thread
	for rows.Next() {
		var t model.Thread
		var pinned, archived int
		var bumpedAt sql.NullTime
		err := rows.Scan(
			&t.ID,
			&t.BoardID,
			&t.Title,
			&bumpedAt,
			&pinned,
			&archived,
			&t.CreatedAt,
			&t.BoardThreadNo,
		)
		if err != nil {
			return nil, err
		}
		t.BumpedAt = effectiveBumpedAt(bumpedAt, t.CreatedAt)
		t.Pinned = pinned != 0
		t.Archived = archived != 0
		threads = append(threads, t)
	}
	return threads, nil
}

func (r *Repository) CountActiveThreadsOnBoard(boardID int) (int, error) {
	var n int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM threads
		WHERE board_id = ? AND COALESCE(archived, 0) = 0
	`, boardID).Scan(&n)
	return n, err
}

func (r *Repository) ArchiveOldestThreads(boardID int, excess int) error {
	if excess <= 0 {
		return nil
	}
	_, err := r.db.Exec(`
		UPDATE threads SET archived = 1
		WHERE id IN (
			SELECT id FROM threads
			WHERE board_id = ? AND COALESCE(archived, 0) = 0
			ORDER BY datetime(COALESCE(bumped_at, created_at)) ASC, id ASC
			LIMIT ?
		)
	`, boardID, excess)
	return err
}

func (r *Repository) CreateThreadWithOP(boardID int, title string, op model.Post) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO threads (board_id, title, bumped_at)
		VALUES (?, ?, datetime('now'))
	`, boardID, title)
	if err != nil {
		return 0, err
	}
	threadID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if op.Text != "" || op.Image != "" {
		_, err = tx.Exec(`
			INSERT INTO posts (
				thread_id, text, image, sage, hidden,
				poster_name, trip_hash, file_hash, mime, file_size, thumb_path
			)
			VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)
		`, threadID, op.Text, op.Image, boolToInt(op.Sage),
			nullStr(op.PosterName), nullStr(op.TripHash),
			nullStr(op.FileHash), nullStr(op.Mime), op.FileSize, nullStr(op.ThumbPath))
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return threadID, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *Repository) GetThreadByID(id int) (model.Thread, error) {
	row := r.db.QueryRow(`
		SELECT
			t.id,
			t.board_id,
			t.title,
			t.bumped_at,
			COALESCE(t.pinned, 0),
			COALESCE(t.archived, 0),
			t.created_at,
			(
				SELECT COUNT(*)
				FROM threads t2
				WHERE t2.board_id = t.board_id
					AND t2.id <= t.id
			) AS board_thread_no
		FROM threads t
		WHERE t.id = ?
	`, id)

	var t model.Thread
	var pinned, archived int
	var bumpedAt sql.NullTime
	err := row.Scan(
		&t.ID,
		&t.BoardID,
		&t.Title,
		&bumpedAt,
		&pinned,
		&archived,
		&t.CreatedAt,
		&t.BoardThreadNo,
	)
	if err != nil {
		return model.Thread{}, err
	}
	t.BumpedAt = effectiveBumpedAt(bumpedAt, t.CreatedAt)
	t.Pinned = pinned != 0
	t.Archived = archived != 0
	return t, nil
}

func (r *Repository) BumpThread(threadID int) error {
	_, err := r.db.Exec(`
		UPDATE threads SET bumped_at = datetime('now') WHERE id = ?
	`, threadID)
	return err
}

func (r *Repository) SetThreadPinned(threadID int, pinned bool) error {
	p := 0
	if pinned {
		p = 1
	}
	_, err := r.db.Exec(`UPDATE threads SET pinned = ? WHERE id = ?`, p, threadID)
	return err
}


