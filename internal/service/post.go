package service

import "github.com/venedicus/imgbrd/internal/model"

func (s *Service) CreatePost(post model.Post) (int64, error) {
	return s.repo.CreatePost(post)
}
