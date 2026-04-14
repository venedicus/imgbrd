package service

import "github.com/venedicus/imgbrd/internal/dto"

func (s *Service) GetDashboard() (dto.GlobalStats, []dto.BoardStat, error) {
	g, err := s.repo.GetGlobalStats()
	if err != nil {
		return dto.GlobalStats{}, nil, err
	}
	boards, err := s.repo.GetBoardStats()
	if err != nil {
		return dto.GlobalStats{}, nil, err
	}
	return g, boards, nil
}
