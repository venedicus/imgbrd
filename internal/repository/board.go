package repository

import (
	"database/sql"
	"errors"

	"github.com/venedicus/imgbrd/internal/model"
)

func (r *Repository) CreateBoard(slug, title string) error {
	_, err := r.db.Exec(`
		INSERT INTO boards (slug, title)
		VALUES (?, ?)
	`, slug, title)
	return err
}

func (r *Repository) UpdateBoardLimits(slug string, maxThreads int, nsfw bool) error {
	n := 0
	if nsfw {
		n = 1
	}
	res, err := r.db.Exec(`
		UPDATE boards SET max_threads = ?, nsfw = ? WHERE slug = ?
	`, maxThreads, n, slug)
	if err != nil {
		return err
	}
	nr, _ := res.RowsAffected()
	if nr == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetBoards() ([]model.Board, error) {
	rows, err := r.db.Query(`
		SELECT id, slug, title, COALESCE(max_threads, 0), COALESCE(nsfw, 0), created_at
		FROM boards
		ORDER BY slug ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []model.Board
	for rows.Next() {
		var b model.Board
		var nsfw int
		if err := rows.Scan(&b.ID, &b.Slug, &b.Title, &b.MaxThreads, &nsfw, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.NSFW = nsfw != 0
		boards = append(boards, b)
	}
	return boards, nil
}

func (r *Repository) GetBoardBySlug(slug string) (model.Board, error) {
	row := r.db.QueryRow(`
		SELECT id, slug, title, COALESCE(max_threads, 0), COALESCE(nsfw, 0), created_at
		FROM boards
		WHERE slug = ?
	`, slug)
	var b model.Board
	var nsfw int
	err := row.Scan(&b.ID, &b.Slug, &b.Title, &b.MaxThreads, &nsfw, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Board{}, err
	}
	b.NSFW = nsfw != 0
	return b, err
}

func (r *Repository) GetBoardByID(id int) (model.Board, error) {
	row := r.db.QueryRow(`
		SELECT id, slug, title, COALESCE(max_threads, 0), COALESCE(nsfw, 0), created_at
		FROM boards
		WHERE id = ?
	`, id)
	var b model.Board
	var nsfw int
	err := row.Scan(&b.ID, &b.Slug, &b.Title, &b.MaxThreads, &nsfw, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Board{}, err
	}
	b.NSFW = nsfw != 0
	return b, err
}
