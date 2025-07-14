package user

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Microsoft/go-winio/pkg/guid"
	request2 "medods/internal/domain/request"
	token2 "medods/internal/domain/token"
	"medods/internal/repository/uow"
	"medods/internal/repository/user"
	"medods/internal/service/token"
)

type UserService struct {
	uow          *uow.UnitOfWork
	tokenService *token.TokenService
}

func NewUService(uow *uow.UnitOfWork, ts *token.TokenService) *UserService {
	return &UserService{uow: uow, tokenService: ts}
}

func (us *UserService) Authorize(ctx context.Context, id string, data request2.RequestData) (token2.TokensPair, error) {
	_, err := guid.FromString(id)
	if err != nil {
		return token2.TokensPair{}, err
	}
	u, err := us.uow.User.Get(ctx, user.WithGuid(id))
	switch {
	case err != nil && errors.Is(err, sql.ErrNoRows):
		u = &user.User{Guid: id}
		err = us.uow.User.Create(ctx, u)
	case err != nil:
		return token2.TokensPair{}, err
	}
	return us.tokenService.Build(ctx, id, data)
}
