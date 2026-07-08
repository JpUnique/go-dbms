package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
)

type TagService struct {
	repo         *repository.TagRepository
	documentRepo *repository.DocumentRepository
	userRepo     *repository.UserRepository
}

func NewTagService(repo *repository.TagRepository, docRepo *repository.DocumentRepository, userRepo *repository.UserRepository) *TagService {
	return &TagService{
		repo:         repo,
		documentRepo: docRepo,
		userRepo:     userRepo,
	}
}

func (s *TagService) GetAll(ctx context.Context, userID, role string) ([]models.Tag, error) {
	_, department, err := resolveDeptScope(ctx, s.userRepo, userID, role)
	if err != nil {
		return nil, fmt.Errorf("tag service get all: %w", err)
	}
	return s.repo.GetAll(ctx, userID, department)
}

func (s *TagService) Create(ctx context.Context, name string, color string) (*models.Tag, error) {
	name = strings.TrimSpace(name)
	return s.repo.Create(ctx, name, color)
}

func (s *TagService) Update(ctx context.Context, id string, name *string, color *string) (*models.Tag, error) {
	return s.repo.Update(ctx, id, name, color)
}

func (s *TagService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TagService) Attach(ctx context.Context, docID, tagID, userID string) error {

	doc, err := s.documentRepo.GetByID(ctx, docID, userID, false, nil)
	if err != nil || doc == nil {
		return fmt.Errorf("not owner")
	}

	return s.repo.Attach(ctx, docID, tagID)
}

func (s *TagService) Detach(ctx context.Context, docID, tagID, userID string) error {

	doc, err := s.documentRepo.GetByID(ctx, docID, userID, false, nil)
	if err != nil || doc == nil {
		return fmt.Errorf("not owner")
	}

	return s.repo.Detach(ctx, docID, tagID)
}

func (s *TagService) GetDocumentsByTag(ctx context.Context, tagID, userID, role string) ([]models.DocumentWithOwner, error) {
	_, department, err := resolveDeptScope(ctx, s.userRepo, userID, role)
	if err != nil {
		return nil, fmt.Errorf("tag service get documents by tag: %w", err)
	}
	return s.repo.GetDocumentsByTag(ctx, tagID, userID, department)
}

func (s *TagService) GetByDocument(
	ctx context.Context,
	docID string,
	userID string,
) ([]models.Tag, error) {

	//  verify document ownership
	doc, err := s.documentRepo.GetByID(ctx, docID, userID, false, nil)
	if err != nil {
		return nil, fmt.Errorf("tag service get document: %w", err)
	}

	if doc == nil {
		return nil, fmt.Errorf("not owner")
	}

	// ✅ fetch tags
	tags, err := s.repo.GetByDocument(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("tag service get by document: %w", err)
	}

	return tags, nil
}
