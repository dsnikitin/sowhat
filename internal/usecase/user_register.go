package usecase

import "context"

type UserService interface {
	RegisterUser(ctx context.Context, id int64, name string) error
}

type RegisterUserUseCase struct {
	us UserService
}

func NewRegisterUserUseCase(us UserService) *RegisterUserUseCase {
	return &RegisterUserUseCase{us: us}
}

func (uc *RegisterUserUseCase) Register(ctx context.Context, id int64, name string) error {
	return uc.us.RegisterUser(ctx, id, name)
}
