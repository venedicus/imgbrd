package repository

import (
	"database/sql"

	"github.com/venedicus/imgbrd/internal/model"
)

func (r *Repository) GetPostsByThreadID(threadID int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT
			p.id,
			p.thread_id,
			p.text,
			p.image,
			COALESCE(p.sage, 0),
			COALESCE(p.hidden, 0),
			COALESCE(p.poster_name, ''),
			COALESCE(p.trip_hash, ''),
			COALESCE(p.file_hash, ''),
			COALESCE(p.mime, ''),
			COALESCE(p.file_size, 0),
			COALESCE(p.thumb_path, ''),
			p.created_at,
			(
				SELECT COUNT(*)
				FROM posts p2
				INNER JOIN threads t2 ON t2.id = p2.thread_id
				WHERE t2.board_id = (SELECT board_id FROM threads WHERE id = ?)
					AND COALESCE(p2.hidden, 0) = 0
					AND p2.id <= p.id
			) AS board_post_no
		FROM posts p
		WHERE p.thread_id = ?
			AND COALESCE(p.hidden, 0) = 0
		ORDER BY p.id ASC
	`, threadID, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		var sage, hidden int
		err := rows.Scan(
			&p.ID,
			&p.ThreadID,
			&p.Text,
			&p.Image,
			&sage,
			&hidden,
			&p.PosterName,
			&p.TripHash,
			&p.FileHash,
			&p.Mime,
			&p.FileSize,
			&p.ThumbPath,
			&p.CreatedAt,
			&p.BoardPostNo,
		)
		if err != nil {
			return nil, err
		}
		p.Sage = sage != 0
		p.Hidden = hidden != 0
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (r *Repository) CreatePost(post model.Post) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO posts (
			thread_id, text, image, sage, hidden,
			poster_name, trip_hash, file_hash, mime, file_size, thumb_path
		)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)
	`, post.ThreadID, post.Text, post.Image, boolToInt(post.Sage),
		nullStr(post.PosterName), nullStr(post.TripHash),
		nullStr(post.FileHash), nullStr(post.Mime), post.FileSize, nullStr(post.ThumbPath))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if !post.Sage {
		_, err = tx.Exec(`
			UPDATE threads SET bumped_at = datetime('now') WHERE id = ?
		`, post.ThreadID)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) FindMediaByHash(hash string) (image string, thumb string, ok bool) {
	if hash == "" {
		return "", "", false
	}
	row := r.db.QueryRow(`
		SELECT image, COALESCE(thumb_path, '')
		FROM posts
		WHERE file_hash = ?
		LIMIT 1
	`, hash)
	var img, th sql.NullString
	if err := row.Scan(&img, &th); err != nil {
		return "", "", false
	}
	return img.String, th.String, img.Valid
}

func (r *Repository) SetPostHidden(postID int, hidden bool) error {
	h := 0
	if hidden {
		h = 1
	}
	_, err := r.db.Exec(`UPDATE posts SET hidden = ? WHERE id = ?`, h, postID)
	return err
}

func (r *Repository) GetPostByID(id int) (model.Post, error) {
	row := r.db.QueryRow(`
		SELECT
			id, thread_id, text, image,
			COALESCE(sage, 0), COALESCE(hidden, 0),
			COALESCE(poster_name, ''), COALESCE(trip_hash, ''),
			COALESCE(file_hash, ''), COALESCE(mime, ''),
			COALESCE(file_size, 0), COALESCE(thumb_path, ''),
			created_at
		FROM posts WHERE id = ?
	`, id)
	var p model.Post
	var sage, hidden int
	err := row.Scan(
		&p.ID, &p.ThreadID, &p.Text, &p.Image,
		&sage, &hidden,
		&p.PosterName, &p.TripHash,
		&p.FileHash, &p.Mime, &p.FileSize, &p.ThumbPath,
		&p.CreatedAt,
	)
	p.Sage = sage != 0
	p.Hidden = hidden != 0
	return p, err
}

func (r *Repository) UpdatePostText(postID int, newText string) (string, error) {
	var old string
	row := r.db.QueryRow(`SELECT text FROM posts WHERE id = ?`, postID)
	if err := row.Scan(&old); err != nil {
		return "", err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO post_edits (post_id, old_text) VALUES (?, ?)`, postID, old); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`UPDATE posts SET text = ? WHERE id = ?`, newText, postID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return old, nil
}
