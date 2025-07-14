package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"medods/api"
	request2 "medods/internal/domain/request"
	"medods/internal/domain/token"
)

func (h *ApiHandler) PostAuthorize(ctx context.Context, request api.PostAuthorizeRequestObject) (api.PostAuthorizeResponseObject, error) {
	fCtx, err := getFiberContext(ctx)
	if err != nil {
		msg := err.Error()
		return api.PostAuthorize500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, err
	}
	ip := fCtx.Locals(IPKey)
	if ip == nil || ip == "" {
		msg := "internal server error"
		return api.PostAuthorize500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, err
	}
	reqData := request2.RequestData{
		UserAgent: request.Params.UserAgent,
		IP:        ip.(string),
	}
	guid := request.Params.Guid
	tokenPairs, err := h.us.Authorize(ctx, guid, reqData)

	if err != nil {
		msg := fmt.Sprintf("error occurred while proccessing request: %s", err.Error())
		return api.PostAuthorize400JSONResponse{N400JSONResponse: api.N400JSONResponse{Message: &msg}}, err
	}
	return api.PostAuthorize200JSONResponse{AccessToken: tokenPairs.AccessToken, RefreshToken: tokenPairs.RefreshToken}, nil
}

func (h *ApiHandler) PostRefresh(ctx context.Context, request api.PostRefreshRequestObject) (api.PostRefreshResponseObject, error) {
	fCtx, err := getFiberContext(ctx)
	if err != nil {
		msg := err.Error()
		return api.PostRefresh500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, err
	}
	ip := fCtx.Locals(IPKey)
	if ip == nil || ip == "" {
		msg := "internal server error"
		return api.PostRefresh500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, err
	}
	reqData := request2.RequestData{
		UserAgent: request.Params.UserAgent,
		IP:        ip.(string),
	}
	var tokenPairs = token.TokensPair{AccessToken: request.Body.AccessToken, RefreshToken: request.Body.RefreshToken}
	err = h.ts.Refresh(ctx, &tokenPairs, reqData)
	if err != nil {
		msg := fmt.Sprintf("error occurred while proccessing request: %s", err.Error())
		return api.PostRefresh400JSONResponse{N400JSONResponse: api.N400JSONResponse{Message: &msg}}, err
	}
	return api.PostRefresh200JSONResponse{AccessToken: tokenPairs.AccessToken, RefreshToken: tokenPairs.RefreshToken}, nil
}

func (h *ApiHandler) GetUserGuid(ctx context.Context, request api.GetUserGuidRequestObject) (api.GetUserGuidResponseObject, error) {
	fCtx, err := getFiberContext(ctx)
	if err != nil {
		msg := err.Error()
		return api.GetUserGuid500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, nil
	}
	userGuid := fCtx.Locals(UserGuidKey)
	userGuidStr := userGuid.(string)
	if userGuid == "" {
		msg := fmt.Sprintf("error occurred while proccessing request")
		return api.GetUserGuid500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, nil
	}
	return api.GetUserGuid200JSONResponse{UserGuid: &userGuidStr}, nil
}

func (h *ApiHandler) PostUserLogout(ctx context.Context, request api.PostUserLogoutRequestObject) (api.PostUserLogoutResponseObject, error) {
	err := h.ts.BlockToken(ctx, request.Params.Authorization)
	if err != nil {
		msg := fmt.Sprintf("error occurred while proccessing request")
		return api.PostUserLogout500JSONResponse{N500JSONResponse: api.N500JSONResponse{Message: &msg}}, err
	}
	return api.PostUserLogout200Response{}, nil
}

func getFiberContext(ctx context.Context) (*fiber.Ctx, error) {
	fiberContext := ctx.Value(FiberContextKey)
	if fiberContext == "" {
		msg := fmt.Sprintf("error occurred while proccessing request")
		return nil, errors.New(msg)
	}
	fCtx, ok := fiberContext.(*fiber.Ctx)
	if !ok {
		msg := fmt.Sprintf("error occurred while proccessing request")
		return nil, errors.New(msg)
	}
	return fCtx, nil
}
