package repository

import (
	"fmt"
	"strings"
)

// SearchPostsFTS runs FTS when scope is posts (caller checks).
func (r *Repository) SearchPostsFTS(f SearchFilter) ([]SearchHit, error) {
	normalizeSearchFilter(&f)
	q := f.Query
	if q == "" {
		return nil, nil
	}
	q = strings.ReplaceAll(q, `"`, ` `)
	if strings.TrimSpace(q) == "" {
		return nil, nil
	}
	parts := strings.Fields(q)
	var match string
	if len(parts) > 0 {
		var b strings.Builder
		for i, p := range parts {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString(`"`)
			b.WriteString(strings.ReplaceAll(p, `"`, ``))
			b.WriteString(`"`)
		}
		match = b.String()
	} else {
		match = q
	}

	boardClause := ""
	args := []any{match}
	if f.BoardID > 0 {
		boardClause = " AND b.id = ?"
		args = append(args, f.BoardID)
	}
	args = append(args, f.Limit)

	order := postSearchOrder(f.Sort)
	query := fmt.Sprintf(`
		SELECT b.slug, p.thread_id, p.id, snippet(posts_fts, 0, '…', '…', 12, 32),
			(
				SELECT COUNT(*)
				FROM posts p2
				INNER JOIN threads t2 ON t2.id = p2.thread_id
				WHERE t2.board_id = t.board_id
					AND COALESCE(p2.hidden, 0) = 0
					AND p2.id <= p.id
			)
		FROM posts_fts
		INNER JOIN posts p ON p.id = posts_fts.rowid
		INNER JOIN threads t ON t.id = p.thread_id
		INNER JOIN boards b ON b.id = t.board_id
		WHERE posts_fts MATCH ?
			AND COALESCE(p.hidden, 0) = 0
			AND COALESCE(t.archived, 0) = 0
			%s
		ORDER BY %s
		LIMIT ?
	`, boardClause, order)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		h.Kind = "post"
		if err := rows.Scan(&h.BoardSlug, &h.ThreadID, &h.PostID, &h.Snippet, &h.BoardPostNo); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// SearchPostsLike is the LIKE fallback for post text search.
func (r *Repository) SearchPostsLike(f SearchFilter) ([]SearchHit, error) {
	normalizeSearchFilter(&f)
	pat := "%" + strings.ReplaceAll(f.Query, "%", "") + "%"
	if f.Query == "" || pat == "%%" {
		return nil, nil
	}

	boardClause := ""
	args := []any{pat}
	if f.BoardID > 0 {
		boardClause = " AND b.id = ?"
		args = append(args, f.BoardID)
	}
	args = append(args, f.Limit)
	order := postSearchOrder(f.Sort)

	query := fmt.Sprintf(`
		SELECT b.slug, p.thread_id, p.id, substr(p.text, 1, 120),
			(
				SELECT COUNT(*)
				FROM posts p2
				INNER JOIN threads t2 ON t2.id = p2.thread_id
				WHERE t2.board_id = t.board_id
					AND COALESCE(p2.hidden, 0) = 0
					AND p2.id <= p.id
			)
		FROM posts p
		INNER JOIN threads t ON t.id = p.thread_id
		INNER JOIN boards b ON b.id = t.board_id
		WHERE p.text LIKE ?
			AND COALESCE(p.hidden, 0) = 0
			AND COALESCE(t.archived, 0) = 0
			%s
		ORDER BY %s
		LIMIT ?
	`, boardClause, order)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		h.Kind = "post"
		if err := rows.Scan(&h.BoardSlug, &h.ThreadID, &h.PostID, &h.Snippet, &h.BoardPostNo); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// SearchThreadsTitle matches thread titles (substring).
func (r *Repository) SearchThreadsTitle(f SearchFilter) ([]SearchHit, error) {
	normalizeSearchFilter(&f)
	pat := "%" + strings.ReplaceAll(f.Query, "%", "") + "%"
	if f.Query == "" || pat == "%%" {
		return nil, nil
	}

	boardClause := ""
	args := []any{pat}
	if f.BoardID > 0 {
		boardClause = " AND b.id = ?"
		args = append(args, f.BoardID)
	}
	args = append(args, f.Limit)
	order := threadSearchOrder(f.Sort)

	query := fmt.Sprintf(`
		SELECT b.slug, t.id, t.title
		FROM threads t
		INNER JOIN boards b ON b.id = t.board_id
		WHERE COALESCE(t.archived, 0) = 0
			AND t.title LIKE ?
			%s
		ORDER BY %s
		LIMIT ?
	`, boardClause, order)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		h.Kind = "thread"
		if err := rows.Scan(&h.BoardSlug, &h.ThreadID, &h.Snippet); err != nil {
			return nil, err
		}
		h.PostID = 0
		h.BoardPostNo = 0
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
