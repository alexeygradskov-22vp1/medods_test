package user

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Microsoft/go-winio/pkg/guid"
	request2 "medods/internal/domain/request"
	token2 "medods/internal/domain/token"
	"medods/internal/repository/user"
	"medods/internal/service/token"
)

type UserService struct {
	userRepo     *user.UserRepository
	tokenService *token.TokenService
}

func NewUService(uRepo *user.UserRepository, ts *token.TokenService) *UserService {
	return &UserService{tokenService: ts, userRepo: uRepo}
}

func (us *UserService) Authorize(ctx context.Context, id string, data request2.RequestData) (token2.TokensPair, error) {
	_, err := guid.FromString(id)
	if err != nil {
		return token2.TokensPair{}, err
	}
	u, err := us.userRepo.GetOneBy(ctx, []user.FieldName{user.GuidFieldName}, []interface{}{id})
	switch {
	case err != nil && errors.Is(err, sql.ErrNoRows):
		u = &user.User{Guid: id}
		err = us.userRepo.Create(ctx, u)
	case err != nil:
		return token2.TokensPair{}, err
	}
	return us.tokenService.Build(ctx, id, data)
}
