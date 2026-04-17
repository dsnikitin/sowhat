package service

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/consts/platform"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/pkg/errors"
)

type UserRepository interface {
	CreateUser(ctx context.Context, externalID, name string, pt platform.Type) error
	GetUserByID(ctx context.Context, userID int64, pl platform.Type) (models.User, error)
	GetUserByExternalID(ctx context.Context, externalID string, pt platform.Type) (models.User, error)
}

type UserService struct {
	r UserRepository
}

func NewUserService(r UserRepository) *UserService {
	return &UserService{r: r}
}

func (s *UserService) RegisterUser(ctx context.Context, externalID, name string, pt platform.Type) error {
	return s.r.CreateUser(ctx, externalID, name, pt)
}

func (s *UserService) GetUserByID(ctx context.Context, userID int64, pl platform.Type) (models.User, error) {
	return s.r.GetUserByID(ctx, userID, pl)
}

func (s *UserService) IdentityUser(ctx context.Context, pt platform.Type, externalUserID string) (int64, error) {
	switch pt {
	case platform.Telegram:
		user, err := s.r.GetUserByExternalID(ctx, externalUserID, pt)
		if err != nil {
			return 0, errors.Wrap(err, "get user")
		}

		return user.ID, nil
	default:
		return 0, errors.Errorf("unsupported platform %s", pt)
	}
}
