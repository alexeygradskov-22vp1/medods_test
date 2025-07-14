package api

import (
	"medods/api"
	"medods/internal/service/token"
	"medods/internal/service/user"
)

type ApiHandler struct {
	api.StrictServerInterface
	ts *token.TokenService
	us *user.UserService
}

func NewApiHandler(ts *token.TokenService, us *user.UserService) *ApiHandler {
	return &ApiHandler{ts: ts, us: us}
}
