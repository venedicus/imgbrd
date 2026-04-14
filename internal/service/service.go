package service

import "github.com/venedicus/imgbrd/internal/repository"

type Service struct {
	repo *repository.Repository
}

func New(r *repository.Repository) *Service {
	return &Service{repo: r}
}