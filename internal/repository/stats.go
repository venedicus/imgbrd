package repository

import (
	"github.com/venedicus/imgbrd/internal/dto"
)

func (r *Repository) GetGlobalStats() (dto.GlobalStats, error) {
	var g dto.GlobalStats
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM boards`).Scan(&g.TotalBoards); err != nil {
		return dto.GlobalStats{}, err
	}
	if err := r.db.QueryRow(`
		SELECT COUNT(*) FROM threads WHERE COALESCE(archived, 0) = 0
	`).Scan(&g.TotalThreads); err != nil {
		return dto.GlobalStats{}, err
	}
	if err := r.db.QueryRow(`
		SELECT COUNT(*) FROM posts WHERE COALESCE(hidden, 0) = 0
	`).Scan(&g.TotalPosts); err != nil {
		return dto.GlobalStats{}, err
	}
	if err := r.db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE COALESCE(hidden, 0) = 0
			AND created_at >= datetime('now', '-1 hour')
	`).Scan(&g.PostsLastHour); err != nil {
		return dto.GlobalStats{}, err
	}
	return g, nil
}

func (r *Repository) GetBoardStats() ([]dto.BoardStat, error) {
	rows, err := r.db.Query(`
		SELECT
			b.slug,
			b.title,
			(SELECT COUNT(*) FROM threads t
			 WHERE t.board_id = b.id AND COALESCE(t.archived, 0) = 0),
			(SELECT COUNT(*) FROM posts p
				INNER JOIN threads t ON t.id = p.thread_id
				WHERE t.board_id = b.id
					AND COALESCE(p.hidden, 0) = 0),
			(SELECT COUNT(*) FROM posts p
				INNER JOIN threads t ON t.id = p.thread_id
				WHERE t.board_id = b.id
					AND COALESCE(p.hidden, 0) = 0
					AND p.created_at >= datetime('now', '-1 hour'))
		FROM boards b
		ORDER BY b.slug ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dto.BoardStat
	for rows.Next() {
		var row dto.BoardStat
		if err := rows.Scan(
			&row.Slug,
			&row.Title,
			&row.ThreadCount,
			&row.PostCount,
			&row.PostsLastHour,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
