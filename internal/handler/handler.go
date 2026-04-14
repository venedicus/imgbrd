package handler

import (
	"github.com/venedicus/imgbrd/internal/config"
	"github.com/venedicus/imgbrd/internal/service"
)

type Handler struct {
	svc *service.Service
	cfg *config.Config
}

func New(s *service.Service, cfg *config.Config) *Handler {
	return &Handler{svc: s, cfg: cfg}
}