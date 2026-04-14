package admin

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/venedicus/imgbrd/internal/dto"
	"github.com/venedicus/imgbrd/internal/service"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Server struct {
	svc   *service.Service
	token string
}

func NewHandler(svc *service.Service, token string) http.Handler {
	s := &Server{svc: svc, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/stats", s.stats)
	mux.HandleFunc("/boards", s.boards)
	mux.HandleFunc("/board-config", s.boardConfig)
	mux.HandleFunc("/ban", s.ban)
	mux.HandleFunc("/bans", s.listBans)
	mux.HandleFunc("/posts/hide", s.hidePost)
	mux.HandleFunc("/posts/unhide", s.unhidePost)
	mux.HandleFunc("/posts/edit", s.editPost)
	mux.HandleFunc("/threads/pin", s.pinThread)
	mux.HandleFunc("/modlog", s.modlogJSON)
	return bearerAuth(token, mux)
}

func bearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "admin token not configured", http.StatusServiceUnavailable)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		got = strings.TrimSpace(got)
		if got == "" {
			got = strings.TrimSpace(r.Header.Get("X-Admin-Token"))
		}
		if got != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "imgbrd-admin"})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	g, boards, err := s.svc.GetDashboard()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Global dto.GlobalStats `json:"global"`
		Boards []dto.BoardStat `json:"boards"`
	}{Global: g, Boards: boards})
}

func (s *Server) boards(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.svc.GetBoards()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var body struct {
			Slug  string `json:"slug"`
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		slug := strings.TrimSpace(body.Slug)
		title := strings.TrimSpace(body.Title)
		if title == "" || !slugRe.MatchString(slug) {
			http.Error(w, "slug (lowercase [a-z0-9-]) and title required", http.StatusBadRequest)
			return
		}
		if err := s.svc.CreateBoard(slug, title); err != nil {
			if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
				http.Error(w, "slug already exists", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"slug": slug, "title": title})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) boardConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Slug        string `json:"slug"`
		MaxThreads  int    `json:"max_threads"`
		NSFW        bool   `json:"nsfw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	if err := s.svc.UpdateBoardLimits(slug, body.MaxThreads, body.NSFW); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("board_config", slug)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ban(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		IP          string `json:"ip"`
		BoardID     *int   `json:"board_id"`
		Reason      string `json:"reason"`
		ExpiresHours *int  `json:"expires_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ip := strings.TrimSpace(body.IP)
	if ip == "" {
		http.Error(w, "ip required", http.StatusBadRequest)
		return
	}
	var exp *time.Time
	if body.ExpiresHours != nil && *body.ExpiresHours > 0 {
		t := time.Now().Add(time.Duration(*body.ExpiresHours) * time.Hour)
		exp = &t
	}
	if err := s.svc.AddBan(ip, body.BoardID, body.Reason, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("ban", ip+" "+body.Reason)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listBans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.svc.ListBans(200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) hidePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PostID int `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PostID <= 0 {
		http.Error(w, "post_id required", http.StatusBadRequest)
		return
	}
	if err := s.svc.HidePost(body.PostID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("hide_post", strconv.Itoa(body.PostID))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unhidePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PostID int `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PostID <= 0 {
		http.Error(w, "post_id required", http.StatusBadRequest)
		return
	}
	if err := s.svc.SetPostHidden(body.PostID, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("unhide_post", strconv.Itoa(body.PostID))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) editPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PostID int    `json:"post_id"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PostID <= 0 {
		http.Error(w, "post_id required", http.StatusBadRequest)
		return
	}
	if _, err := s.svc.UpdatePostText(body.PostID, body.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("edit_post", strconv.Itoa(body.PostID))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pinThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ThreadID int  `json:"thread_id"`
		Pinned   bool `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ThreadID <= 0 {
		http.Error(w, "thread_id required", http.StatusBadRequest)
		return
	}
	if err := s.svc.SetThreadPinned(body.ThreadID, body.Pinned); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.svc.AddModLog("pin_thread", strconv.Itoa(body.ThreadID))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) modlogJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.svc.ListModLog(500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}
