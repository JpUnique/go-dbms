package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/repository"
)

type ReportService struct {
	repo *repository.ReportRepository
}

func NewReportService(repo *repository.ReportRepository) *ReportService {
	return &ReportService{repo: repo}
}

// GetReport defaults to "today" when periodParam is empty, and rejects any
// value that isn't one of repository.ReportPeriods (the only keys ever
// interpolated into the underlying SQL).
//
// Every user generates their own report by default (userID is always
// scoped to the caller); allUsers switches to the system-wide view.
func (s *ReportService) GetReport(ctx context.Context, periodParam string, userID string, allUsers bool) (map[string]interface{}, error) {

	period := periodParam
	if period == "" {
		period = "today"
	}
	if _, ok := repository.ReportPeriods[period]; !ok {
		return nil, fmt.Errorf("invalid period: must be one of today, yesterday, week, month")
	}

	scopeUserID := userID
	if allUsers {
		scopeUserID = ""
	}

	data, err := s.repo.GetReport(ctx, period, scopeUserID)
	if err != nil {
		return nil, fmt.Errorf("report service: %w", err)
	}

	return data, nil
}
