package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/repository"
)

type StatsService struct {
	repo *repository.StatsRepository
}

func NewStatsService(repo *repository.StatsRepository) *StatsService {
	return &StatsService{repo: repo}
}

func (s *StatsService) GetDashboard(
	ctx context.Context,
	userID string,
	role string,
) (map[string]interface{}, error) {

	// TEMP-NO-ROLES: was `isAdmin := role == "admin"` — everyone sees
	// system-wide dashboard stats while testing. Restore the line above
	// once role-based access is reintroduced.
	_ = role
	isAdmin := true

	data, err := s.repo.GetDashboard(ctx, userID, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("stats service dashboard: %w", err)
	}

	return data, nil
}

func (s *StatsService) GetActivity(
	ctx context.Context,
	userID string,
	role string,
	period string,
) ([]map[string]interface{}, error) {

	// TEMP-NO-ROLES: was `isAdmin := role == "admin"` — restore once
	// role-based access is reintroduced.
	_ = role
	isAdmin := true

	return s.repo.GetActivity(ctx, userID, isAdmin, period)
}
