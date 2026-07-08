package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
)

type AuditService struct {
	repo *repository.AuditRepository
}

func NewAuditService(repo *repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) GetAll(
	ctx context.Context,
	userID string,
	role string,
	userFilter string,
	resourceType string,
	resourceID string,
	action string,
	limit int,
	offset int,
) ([]models.AuditLog, error) {

	isAdmin := role == "admin"

	logs, err := s.repo.GetAll(
		ctx,
		userID,
		isAdmin,
		userFilter,
		resourceType,
		resourceID,
		action,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("audit service get all: %w", err)
	}

	return logs, nil
}

func (s *AuditService) Delete(
	ctx context.Context,
	before string,
) error {

	return s.repo.Delete(ctx, before)
}
