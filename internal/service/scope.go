package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/repository"
)

// resolveDeptScope turns a JWT role into the (isAdmin, department) pair
// repos need to widen an owner_id-only check to "own OR same department":
// admins bypass department scoping entirely (department stays nil, isAdmin
// does the work); everyone else gets their own department looked up.
// department stays nil if the user has none set. Shared by
// DocumentService, DocumentVersionService, and TagService.
func resolveDeptScope(ctx context.Context, userRepo *repository.UserRepository, userID, role string) (isAdmin bool, department *string, err error) {
	if role == "admin" {
		return true, nil, nil
	}

	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, nil, fmt.Errorf("resolve dept scope: %w", err)
	}
	if user == nil || user.Department == nil || *user.Department == "" {
		return false, nil, nil
	}

	return false, user.Department, nil
}
