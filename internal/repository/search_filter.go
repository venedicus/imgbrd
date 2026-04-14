package repository

import "strings"

// SearchFilter controls board scope, sort order, and posts vs thread titles.
type SearchFilter struct {
	Query   string
	BoardID int // 0 = all boards
	Sort    string
	Scope   string // "posts" or "threads"
	Limit   int
}

// SearchHit is one row from post or thread search.
type SearchHit struct {
	Kind        string
	BoardSlug   string
	ThreadID    int
	PostID      int
	BoardPostNo int
	Snippet     string
}

func normalizeSearchFilter(f *SearchFilter) {
	f.Query = strings.TrimSpace(f.Query)
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	switch f.Sort {
	case "old", "bump":
	default:
		f.Sort = "new"
	}
	if f.Scope != "threads" {
		f.Scope = "posts"
	}
}

func postSearchOrder(sort string) string {
	switch sort {
	case "old":
		return "p.id ASC"
	case "bump":
		return "datetime(COALESCE(t.bumped_at, t.created_at)) DESC, p.id DESC"
	default:
		return "p.id DESC"
	}
}

func threadSearchOrder(sort string) string {
	switch sort {
	case "old":
		return "t.id ASC"
	case "bump":
		return "datetime(COALESCE(t.bumped_at, t.created_at)) DESC, t.id DESC"
	default:
		return "t.id DESC"
	}
}
