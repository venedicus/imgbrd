package main

import (
	"database/sql"
	"log"
	"net/http"
	_ "net/http/pprof"

	_ "modernc.org/sqlite"

	"github.com/dchest/captcha"
	"github.com/venedicus/imgbrd/internal/admin"
	"github.com/venedicus/imgbrd/internal/config"
	"github.com/venedicus/imgbrd/internal/db"
	"github.com/venedicus/imgbrd/internal/handler"
	"github.com/venedicus/imgbrd/internal/repository"
	"github.com/venedicus/imgbrd/internal/service"
)

func main() {
	cfg := config.Load()

	sqlDB, err := sql.Open("sqlite", "file:./data.db")
	if err != nil {
		log.Fatal("db open error:", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("db ping error:", err)
	}

	if err := db.Migrate(sqlDB); err != nil {
		log.Fatal("db migrate error:", err)
	}

	log.Println("database connected")

	repo := repository.New(sqlDB)
	svc := service.New(repo)
	h := handler.New(svc, &cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/board/", h.BoardCatalog)
	mux.HandleFunc("/set-theme", h.SetTheme)
	mux.HandleFunc("/thread/create", h.CreateThread)
	mux.HandleFunc("/thread/view", h.ViewThread)
	mux.HandleFunc("/post/create", h.CreatePost)
	mux.HandleFunc("/api/board/", h.APIBoard)
	mux.HandleFunc("/api/thread", h.APIThread)
	mux.HandleFunc("/rss/board/", h.RSSBoard)
	mux.HandleFunc("/search", h.SearchPage)
	mux.HandleFunc("/thread/export", h.ThreadExport)

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))

	if cfg.PprofAddr != "" {
		go func() {
			log.Println("pprof listening on http://" + cfg.PprofAddr)
			if err := http.ListenAndServe(cfg.PprofAddr, http.DefaultServeMux); err != nil {
				log.Fatal("pprof error:", err)
			}
		}()
	}

	if cfg.AdminAddr != "" {
		if cfg.AdminToken == "" {
			log.Fatal("IMGBRD_ADMIN_TOKEN is required when IMGBRD_ADMIN_ADDR is set")
		}
		ah := admin.NewHandler(svc, cfg.AdminToken)
		go func() {
			log.Println("imgbrd admin API listening on http://" + cfg.AdminAddr)
			if err := http.ListenAndServe(cfg.AdminAddr, ah); err != nil {
				log.Fatal("admin server error:", err)
			}
		}()
	}

	log.Println("imgbrd public site at http://localhost" + cfg.PublicAddr)
	if err := http.ListenAndServe(cfg.PublicAddr, mux); err != nil {
		log.Fatal("server error:", err)
	}
}
