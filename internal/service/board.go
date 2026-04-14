package service

import (
	"database/sql"
	"errors"

	"github.com/venedicus/imgbrd/internal/model"
)

func (s *Service) GetBoards() ([]model.Board, error) {
	return s.repo.GetBoards()
}

func (s *Service) GetBoardByID(id int) (model.Board, error) {
	return s.repo.GetBoardByID(id)
}

func (s *Service) GetBoardBySlug(slug string) (model.Board, error) {
	b, err := s.repo.GetBoardBySlug(slug)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Board{}, err
	}
	return b, err
}

func (s *Service) CreateBoard(slug, title string) error {
	return s.repo.CreateBoard(slug, title)
}

func (s *Service) GetBoardThreads(slug string) (model.Board, []model.Thread, error) {
	board, err := s.GetBoardBySlug(slug)
	if err != nil {
		return model.Board{}, nil, err
	}
	threads, err := s.repo.GetThreadsByBoardID(board.ID)
	if err != nil {
		return model.Board{}, nil, err
	}
	return board, threads, nil
}
