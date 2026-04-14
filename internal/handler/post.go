package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/venedicus/imgbrd/internal/model"
	"github.com/venedicus/imgbrd/internal/ratelimit"
	"github.com/venedicus/imgbrd/internal/util"
	"github.com/venedicus/imgbrd/internal/webhook"
)

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	threadID, err := strconv.Atoi(r.FormValue("thread_id"))
	if err != nil {
		http.Error(w, "invalid thread id", http.StatusBadRequest)
		return
	}

	th, err := h.svc.GetThreadByID(threadID)
	if err != nil {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	ip := ratelimit.ClientIP(r)
	if banned, _ := h.svc.IsBanned(ip, th.BoardID); banned {
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

	text := strings.TrimSpace(r.FormValue("text"))
	sage := r.FormValue("sage") == "on"
	name := strings.TrimSpace(r.FormValue("name"))
	_, tripFull := util.TripFromPassword(r.FormValue("trip"))

	med, err := h.processMediaField(r, "image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imagePath := ""
	var hash, mime string
	var size int64
	var thumb string
	if med != nil {
		imagePath = med.URL
		hash = med.Hash
		mime = med.Mime
		size = med.Size
		thumb = med.Thumb
	}

	if text == "" && imagePath == "" {
		http.Error(w, "нужен текст или файл", http.StatusBadRequest)
		return
	}

	post := model.Post{
		ThreadID:   threadID,
		Text:       text,
		Image:      imagePath,
		Sage:       sage,
		PosterName: name,
		TripHash:   tripFull,
		FileHash:   hash,
		Mime:       mime,
		FileSize:   size,
		ThumbPath:  thumb,
	}

	pid, err := h.svc.CreatePost(post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	board, _ := h.svc.GetBoardByID(th.BoardID)
	prev := text
	if len([]rune(prev)) > 80 {
		prev = string([]rune(prev)[:80])
	}
	webhook.PostJSON(h.cfg.WebhookURL, h.cfg.WebhookSecret, webhook.PostEvent{
		Event:       "post.created",
		BoardSlug:   board.Slug,
		ThreadID:    threadID,
		PostID:      pid,
		TextPreview: prev,
	})

	ratelimit.RecordPost(r)
	http.Redirect(w, r, "/thread/view?id="+strconv.Itoa(threadID), http.StatusSeeOther)
}
