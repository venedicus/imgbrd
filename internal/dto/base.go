package dto

import "github.com/venedicus/imgbrd/internal/model"

// Base is embedded in every public HTML page for shared chrome (nav, search bar).
type Base struct {
	SiteTitle        string
	Theme            string
	NavBoards        []model.Board
	SearchQuery      string
	CurrentBoardSlug string
}
