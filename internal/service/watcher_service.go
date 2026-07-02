package service

import (
	"context"

	"github.com/JpUnique/go-dbms/internal/repository"
)

type WatcherService struct {
	repo *repository.WatcherRepository
}

func NewWatcherService(repo *repository.WatcherRepository) *WatcherService {
	return &WatcherService{repo: repo}
}

func (s *WatcherService) Toggle(ctx context.Context, documentID, userID string) (bool, error) {
	return s.repo.Toggle(ctx, documentID, userID)
}

func (s *WatcherService) Status(ctx context.Context, documentID, userID string) (bool, int, error) {
	watching, err := s.repo.IsWatching(ctx, documentID, userID)
	if err != nil {
		return false, 0, err
	}
	count, err := s.repo.WatcherCount(ctx, documentID)
	return watching, count, err
}

func (s *WatcherService) WatcherIDs(ctx context.Context, documentID string) ([]string, error) {
	return s.repo.WatcherUserIDs(ctx, documentID)
}
