package dto

import (
	"html/template"
	"strings"

	"github.com/venedicus/imgbrd/internal/model"
)

type PostView struct {
	model.Post
	BodyHTML   template.HTML
	ThumbURL   string
	PosterLine string
	IsVideo    bool
}

func PosterLine(name, tripHashFull string) string {
	var b strings.Builder
	if name != "" {
		b.WriteString(name)
	}
	if tripHashFull != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString("!")
		if len(tripHashFull) >= 8 {
			b.WriteString(tripHashFull[:8])
		} else {
			b.WriteString(tripHashFull)
		}
	}
	return strings.TrimSpace(b.String())
}

func ThumbForPost(p model.Post) string {
	if p.ThumbPath != "" {
		return p.ThumbPath
	}
	return p.Image
}

type ThreadPage struct {
	Base
	Thread    model.Thread
	BoardSlug string
	Posts     []PostView
	CaptchaID string
}

type BoardPage struct {
	Base
	Board     model.Board
	Threads   []model.Thread
	CaptchaID string
}

// SearchPage is the /search results view (quick bar lives in base.html).
type SearchPage struct {
	Base
	Query     string
	Advanced  bool
	BoardSlug string
	Sort      string
	Scope     string
	Hits      []SearchHitView
}

// SearchHitView is one row on the search results page.
type SearchHitView struct {
	Kind        string
	BoardSlug   string
	ThreadID    int
	PostID      int
	BoardPostNo int
	Snippet     string
}
