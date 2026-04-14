package handler

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/venedicus/imgbrd/internal/dto"
	"github.com/venedicus/imgbrd/internal/repository"
	"github.com/venedicus/imgbrd/internal/util"
)

func (h *Handler) APIBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimPrefix(r.URL.Path, "/api/board/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	board, threads, err := h.svc.GetBoardThreads(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"slug":    board.Slug,
		"title":   board.Title,
		"nsfw":    board.NSFW,
		"threads": threads,
	})
}

func (h *Handler) APIThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	thread, slug, posts, err := h.svc.GetThreadWithPosts(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"board_slug": slug,
		"thread":     thread,
		"posts":      posts,
	})
}

type rssChannel struct {
	XMLName xml.Name        `xml:"rss"`
	Version string          `xml:"version,attr"`
	Ch      rssChannelInner `xml:"channel"`
}

type rssChannelInner struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func (h *Handler) RSSBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimPrefix(r.URL.Path, "/rss/board/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	board, threads, err := h.svc.GetBoardThreads(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	base := "http://" + r.Host
	items := make([]rssItem, 0, len(threads))
	for _, t := range threads {
		items = append(items, rssItem{
			Title:   t.Title,
			Link:    base + "/thread/view?id=" + strconv.Itoa(t.ID),
			GUID:    "thread-" + strconv.Itoa(t.ID),
			PubDate: t.BumpedAt.UTC().Format(time.RFC1123Z),
		})
	}
	out := rssChannel{Version: "2.0", Ch: rssChannelInner{
		Title:       board.Title,
		Link:        base + "/board/" + board.Slug,
		Description: board.Title,
		Items:       items,
	}}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(out)
}

func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	boardSlug := strings.TrimSpace(r.URL.Query().Get("board"))
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	advanced := r.URL.Query().Get("advanced") == "1"

	if sort != "old" && sort != "bump" {
		sort = "new"
	}
	if scope != "threads" {
		scope = "posts"
	}

	boardID := 0
	if boardSlug != "" {
		b, err := h.svc.GetBoardBySlug(boardSlug)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err == nil {
			boardID = b.ID
		}
	}

	filter := repository.SearchFilter{
		Query:   q,
		BoardID: boardID,
		Sort:    sort,
		Scope:   scope,
		Limit:   80,
	}

	var raw []repository.SearchHit
	if q != "" {
		var err error
		raw, err = h.svc.Search(filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	hits := make([]dto.SearchHitView, 0, len(raw))
	for _, x := range raw {
		hits = append(hits, dto.SearchHitView{
			Kind:        x.Kind,
			BoardSlug:   x.BoardSlug,
			ThreadID:    x.ThreadID,
			PostID:      x.PostID,
			BoardPostNo: x.BoardPostNo,
			Snippet:     x.Snippet,
		})
	}

	if advanced || boardSlug != "" || sort != "new" || scope != "posts" {
		advanced = true
	}

	page := dto.SearchPage{
		Base:      h.baseLayout(r, q, boardSlug),
		Query:     q,
		Advanced:  advanced,
		BoardSlug: boardSlug,
		Sort:      sort,
		Scope:     scope,
		Hits:      hits,
	}

	tmpl, err := template.ParseFiles("templates/search.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = tmpl.ExecuteTemplate(w, "search.html", page)
}

func (h *Handler) ThreadExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	thread, slug, posts, err := h.svc.GetThreadWithPosts(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="thread-%d.zip"`, id))

	zw := zip.NewWriter(w)
	defer zw.Close()

	manifest := fmt.Sprintf(`{"board_slug":%q,"thread_id":%d,"title":%q,"exported":%q,"posts":%d}`,
		slug, thread.ID, thread.Title, time.Now().UTC().Format(time.RFC3339), len(posts))
	_ = zipString(zw, "manifest.json", manifest)

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><meta charset=utf-8><title>")
	sb.WriteString(html.EscapeString(thread.Title))
	sb.WriteString("</title><h1>")
	sb.WriteString(html.EscapeString(thread.Title))
	sb.WriteString("</h1>")
	for _, p := range posts {
		body := util.RenderPostBody(p.Text)
		sb.WriteString(fmt.Sprintf(`<article id="p%d"><header>№%d</header>`, p.BoardPostNo, p.BoardPostNo))
		if p.PosterName != "" || p.TripHash != "" {
			sb.WriteString("<div>" + html.EscapeString(dto.PosterLine(p.PosterName, p.TripHash)) + "</div>")
		}
		sb.WriteString(string(body))
		if p.Image != "" {
			sb.WriteString(`<p><a href="` + html.EscapeString(p.Image) + `">file</a></p>`)
		}
		sb.WriteString("</article>")
	}
	_ = zipString(zw, "thread.html", sb.String())

	for _, p := range posts {
		if p.Image == "" {
			continue
		}
		path := strings.TrimPrefix(p.Image, "/")
		path = filepath.FromSlash(path)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := zw.Create(filepath.Base(path))
		if err != nil {
			continue
		}
		_, _ = f.Write(data)
	}
}

func zipString(zw *zip.Writer, name, s string) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, s)
	return err
}
