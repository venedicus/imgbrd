package service

import (
	"strings"
	"time"

	"github.com/venedicus/imgbrd/internal/model"
	"github.com/venedicus/imgbrd/internal/repository"
)

func (s *Service) FindDedupMedia(hash string) (image, thumb string, ok bool) {
	return s.repo.FindMediaByHash(hash)
}

func (s *Service) IsBanned(ip string, boardID int) (bool, error) {
	return s.repo.IsBanned(ip, boardID)
}

func (s *Service) Search(f repository.SearchFilter) ([]repository.SearchHit, error) {
	if strings.TrimSpace(f.Query) == "" {
		return nil, nil
	}
	if f.Scope == "threads" {
		return s.repo.SearchThreadsTitle(f)
	}
	hits, err := s.repo.SearchPostsFTS(f)
	if err != nil || len(hits) == 0 {
		return s.repo.SearchPostsLike(f)
	}
	return hits, nil
}

func (s *Service) AddBan(ip string, boardID *int, reason string, exp *time.Time) error {
	return s.repo.AddBan(ip, boardID, reason, exp)
}

func (s *Service) ListBans(limit int) ([]model.Ban, error) {
	return s.repo.ListBans(limit)
}

func (s *Service) AddModLog(action, detail string) error {
	return s.repo.AddModLog(action, detail)
}

func (s *Service) ListModLog(limit int) ([]model.ModLogEntry, error) {
	return s.repo.ListModLog(limit)
}

func (s *Service) HidePost(postID int) error {
	return s.repo.SetPostHidden(postID, true)
}

func (s *Service) SetPostHidden(postID int, hidden bool) error {
	return s.repo.SetPostHidden(postID, hidden)
}

func (s *Service) UpdateBoardLimits(slug string, maxThreads int, nsfw bool) error {
	return s.repo.UpdateBoardLimits(slug, maxThreads, nsfw)
}

func (s *Service) SetThreadPinned(threadID int, pinned bool) error {
	return s.repo.SetThreadPinned(threadID, pinned)
}

func (s *Service) UpdatePostText(postID int, newText string) (old string, err error) {
	return s.repo.UpdatePostText(postID, newText)
}

func (s *Service) GetPostByID(id int) (model.Post, error) {
	return s.repo.GetPostByID(id)
}

func (s *Service) GetThreadByID(id int) (model.Thread, error) {
	return s.repo.GetThreadByID(id)
}
