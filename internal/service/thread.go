package service

import (
	"database/sql"
	"errors"

	"github.com/venedicus/imgbrd/internal/model"
)

func (s *Service) CreateThreadOnBoard(boardSlug, title string, op model.Post) (int64, error) {
	board, err := s.repo.GetBoardBySlug(boardSlug)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err != nil {
		return 0, err
	}
	tid, err := s.repo.CreateThreadWithOP(board.ID, title, op)
	if err != nil {
		return 0, err
	}
	if board.MaxThreads > 0 {
		if err := s.PruneBoardThreads(board.ID, board.MaxThreads); err != nil {
			return tid, err
		}
	}
	return tid, nil
}

func (s *Service) PruneBoardThreads(boardID, maxThreads int) error {
	if maxThreads <= 0 {
		return nil
	}
	n, err := s.repo.CountActiveThreadsOnBoard(boardID)
	if err != nil {
		return err
	}
	if n > maxThreads {
		return s.repo.ArchiveOldestThreads(boardID, n-maxThreads)
	}
	return nil
}

func (s *Service) GetThreadWithPosts(threadID int) (model.Thread, string, []model.Post, error) {
	thread, err := s.repo.GetThreadByID(threadID)
	if err != nil {
		return model.Thread{}, "", nil, err
	}

	board, err := s.repo.GetBoardByID(thread.BoardID)
	if err != nil {
		return model.Thread{}, "", nil, err
	}

	posts, err := s.repo.GetPostsByThreadID(threadID)
	if err != nil {
		return model.Thread{}, "", nil, err
	}

	return thread, board.Slug, posts, nil
}
