package service

import "context"

type UserRepository interface {
	Create(ctx context.Context, id int64, name string) error
}

type UserService struct {
	r UserRepository
}

func NewUserService(r UserRepository) *UserService {
	return &UserService{r: r}
}

func (us *UserService) RegisterUser(ctx context.Context, id int64, name string) error {
	return us.r.Create(ctx, id, name)
}
