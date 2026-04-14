package handler

import (
	"net/http"

	"github.com/venedicus/imgbrd/internal/dto"
	"github.com/venedicus/imgbrd/internal/theme"
)

func (h *Handler) baseLayout(r *http.Request, searchQ, currentBoardSlug string) dto.Base {
	navBoards, _ := h.svc.GetBoards()
	return dto.Base{
		SiteTitle:        h.cfg.SiteTitle,
		Theme:            theme.Resolve(r, h.cfg.DefaultTheme),
		NavBoards:        navBoards,
		SearchQuery:      searchQ,
		CurrentBoardSlug: currentBoardSlug,
	}
}
