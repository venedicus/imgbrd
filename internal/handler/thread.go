package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/venedicus/imgbrd/internal/dto"
	"github.com/venedicus/imgbrd/internal/model"
	"github.com/venedicus/imgbrd/internal/ratelimit"
	"github.com/venedicus/imgbrd/internal/theme"
	"github.com/venedicus/imgbrd/internal/util"
	"github.com/venedicus/imgbrd/internal/webhook"
)

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	global, boards, err := h.svc.GetDashboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := dto.HomePage{
		Base:   h.baseLayout(r, "", ""),
		Global: global,
		Boards: boards,
	}

	tmpl, err := template.ParseFiles("templates/index.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) SetTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("theme")
	if q == "" {
		q = r.URL.Query().Get("name")
	}
	id := theme.Sanitize(theme.NameFromQuery(q), h.cfg.DefaultTheme)
	theme.SetCookie(w, id)

	redir := "/"
	if ref := r.Header.Get("Referer"); ref != "" {
		u, err := url.Parse(ref)
		if err == nil && strings.EqualFold(u.Host, r.Host) && u.Path != "" {
			redir = u.Path
			if u.RawQuery != "" {
				redir += "?" + u.RawQuery
			}
		}
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}

func (h *Handler) BoardCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/board/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}

	board, threads, err := h.svc.GetBoardThreads(slug)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := dto.BoardPage{
		Base:      h.baseLayout(r, "", board.Slug),
		Board:     board,
		Threads:   threads,
		CaptchaID: captcha.New(),
	}

	tmpl, err := template.ParseFiles("templates/board.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "board.html", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) CreateThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	boardSlug := r.FormValue("board_slug")
	if boardSlug == "" {
		http.Error(w, "board_slug required", http.StatusBadRequest)
		return
	}

	board, err := h.svc.GetBoardBySlug(boardSlug)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "board not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ip := ratelimit.ClientIP(r)
	if banned, _ := h.svc.IsBanned(ip, board.ID); banned {
		http.Error(w, "запрещено", http.StatusForbidden)
		return
	}

	if ok, wait := ratelimit.AllowPost(r); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Round(time.Second).Seconds())+1))
		http.Error(w, "подождите несколько секунд", http.StatusTooManyRequests)
		return
	}

	if !captcha.VerifyString(r.FormValue("captcha_id"), r.FormValue("captcha_solution")) {
		http.Error(w, "неверная капча", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	sage := r.FormValue("sage") == "on"
	name := strings.TrimSpace(r.FormValue("name"))
	_, tripFull := util.TripFromPassword(r.FormValue("trip"))

	med, err := h.processMediaField(r, "image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	op := model.Post{
		Text:       text,
		Sage:       sage,
		PosterName: name,
		TripHash:   tripFull,
	}
	if med != nil {
		op.Image = med.URL
		op.FileHash = med.Hash
		op.Mime = med.Mime
		op.FileSize = med.Size
		op.ThumbPath = med.Thumb
	}

	tid, err := h.svc.CreateThreadOnBoard(boardSlug, title, op)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "board not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prev := text
	if len([]rune(prev)) > 80 {
		prev = string([]rune(prev)[:80])
	}
	webhook.PostJSON(h.cfg.WebhookURL, h.cfg.WebhookSecret, webhook.PostEvent{
		Event:       "thread.created",
		BoardSlug:   boardSlug,
		ThreadID:    int(tid),
		PostID:      0,
		TextPreview: prev,
	})

	ratelimit.RecordPost(r)
	http.Redirect(w, r, "/thread/view?id="+strconv.FormatInt(tid, 10), http.StatusSeeOther)
}

func (h *Handler) ViewThread(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	threadID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid thread id", http.StatusBadRequest)
		return
	}

	thread, boardSlug, posts, err := h.svc.GetThreadWithPosts(threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pviews := make([]dto.PostView, 0, len(posts))
	for _, p := range posts {
		pviews = append(pviews, dto.PostView{
			Post:       p,
			BodyHTML:   util.RenderPostBody(p.Text),
			ThumbURL:   dto.ThumbForPost(p),
			PosterLine: dto.PosterLine(p.PosterName, p.TripHash),
			IsVideo:    strings.HasPrefix(p.Mime, "video/"),
		})
	}

	page := dto.ThreadPage{
		Base:      h.baseLayout(r, "", boardSlug),
		Thread:    thread,
		BoardSlug: boardSlug,
		Posts:     pviews,
		CaptchaID: captcha.New(),
	}

	tmpl, err := template.ParseFiles("templates/thread.html", "templates/base.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "thread.html", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
