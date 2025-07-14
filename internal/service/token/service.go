package token

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"medods/internal/client/external"
	request "medods/internal/domain/request"
	"medods/internal/domain/token"
	"medods/internal/repository/blacklist"
	token2 "medods/internal/repository/token"
	"medods/internal/repository/uow"
	"time"
)

const (
	Issuer string = "medods_test"
	JwtId  string = "JWTID"
)

type TokenService struct {
	uow *uow.UnitOfWork
	c   *external.ExternalServiceClient
}

func NewTService(uow *uow.UnitOfWork, c *external.ExternalServiceClient) *TokenService {
	return &TokenService{uow: uow, c: c}
}

var alg = jwt.NewHS512([]byte("secret_phrase"))

func buildAccessToken(userGuid string, key string) (string, error) {
	timeNow := time.Now()
	pl := token.AccessTokenPayload{
		Payload: jwt.Payload{
			Issuer:         Issuer,
			Subject:        userGuid,
			Audience:       jwt.Audience{"localhost:8080"},
			ExpirationTime: jwt.NumericDate(timeNow.Add(24 * time.Hour)),
			NotBefore:      jwt.NumericDate(timeNow.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(timeNow),
			JWTID:          JwtId,
		},
		Key: key,
	}
	buildedToken, err := jwt.Sign(pl, alg)
	if err != nil {
		return "", err
	}
	return string(buildedToken), nil
}

func buildRefreshToken(key string) (refresh string, err error) {

	pl := token.RefreshTokenPayload{Time: time.Now().String(), Key: key}
	plBytes, err := json.Marshal(pl)
	if err != nil {
		return "", err
	}
	encodedLen := base64.RawURLEncoding.EncodedLen(len(plBytes))
	refreshBytes := make([]byte, encodedLen)
	base64.RawURLEncoding.Encode(refreshBytes, plBytes)
	refresh = string(refreshBytes)
	return
}

func (ts *TokenService) Refresh(ctx context.Context, tokenPayload *token.TokensPair, data request.RequestData) (err error) {
	tokenBytes := []byte(tokenPayload.AccessToken)
	var tokenPl token.AccessTokenPayload
	_, err = jwt.Verify(tokenBytes, alg, &tokenPl)
	if err != nil {
		return err
	}
	var refreshTokenPl token.RefreshTokenPayload
	err = decodeRefreshToken(&refreshTokenPl, tokenPayload.RefreshToken)
	if err != nil {
		return err
	}
	if refreshTokenPl.Key != tokenPl.Key {
		return errors.New("invalid token")
	}
	var access, refresh string
	err = ts.uow.WithTransaction(ctx, func(tx *uow.UnitOfWork) error {
		oldTokens, txErr := tx.Token.GetMany(ctx, token2.WithUserGuid(tokenPl.Subject), token2.WithActive(true))
		if txErr != nil {
			return txErr
		}
		genUuid, txErr := uuid.NewUUID()
		if txErr != nil {
			return txErr
		}
		validRefresh, txErr := findActiveRefreshTokenFromDBByRefreshToken(oldTokens, tokenPayload.RefreshToken)
		if txErr != nil {
			return txErr
		}
		if validRefresh.UserAgent != data.UserAgent {
			txErr = ts.BlockToken(ctx, tokenPayload.AccessToken)
			return errors.New("user agent mismatch")
		}
		if validRefresh.IP != data.IP {
			txErr = ts.c.SendWebhook("dummy_url", map[string]string{"warn": fmt.Sprintf("changed id from %s to %s", validRefresh.IP, data.IP)})
			if txErr != nil {
				log.Printf("error while sending webhook: %s", txErr.Error())
			}
		}
		access, txErr = buildAccessToken(validRefresh.UserGuid, genUuid.String())
		if txErr != nil {
			return txErr
		}
		refresh, err = buildRefreshToken(genUuid.String())
		if txErr != nil {
			return txErr
		}
		bcrypted, txErr := processForStorageInDatabase(refresh)
		if txErr != nil {
			return txErr
		}
		u := token2.NewUpdate(token2.WithUpdateActive(false))
		txErr = tx.Token.Update(ctx, u, token2.WithUserGuid(validRefresh.UserGuid))
		if txErr != nil {
			return txErr
		}
		created := &token2.Token{RefreshToken: bcrypted, UserGuid: validRefresh.UserGuid, Active: true, UserAgent: data.UserAgent, IP: data.IP}
		txErr = tx.Token.Create(ctx, created)
		if txErr != nil {
			return txErr
		}
		return nil
	})
	tokenPayload.AccessToken = access
	tokenPayload.RefreshToken = refresh
	return err
}

func processForStorageInDatabase(token string) (string, error) {
	refreshBytes := []byte(token)
	sh := sha256.Sum256(refreshBytes)
	hashed, err := bcrypt.GenerateFromPassword(sh[:], bcrypt.DefaultCost)
	return string(hashed), err
}

func (ts *TokenService) Valid(accessToken string, dst *token.AccessTokenPayload) error {
	tokenBytes := []byte(accessToken)
	_, err := jwt.Verify(tokenBytes, alg, &dst)
	if err != nil {
		return err
	}
	now := time.Now()
	if now.After(dst.ExpirationTime.Time) {
		return errors.New("token expired")
	}
	return nil
}

func decodeRefreshToken(pl *token.RefreshTokenPayload, refresh string) error {
	decoded, err := base64.RawURLEncoding.DecodeString(refresh)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, pl)
}

func (ts *TokenService) Build(ctx context.Context, id string, data request.RequestData) (token.TokensPair, error) {
	var refresh string
	genUuid, err := uuid.NewUUID()
	if err != nil {
		return token.TokensPair{}, err
	}
	access, err := buildAccessToken(id, genUuid.String())
	if err != nil {
		return token.TokensPair{}, err
	}
	err = ts.uow.WithTransaction(ctx, func(tx *uow.UnitOfWork) error {
		refreshResult, txErr := buildRefreshToken(genUuid.String())
		if txErr != nil {
			return txErr
		}
		bcryptedPass, txErr := processForStorageInDatabase(refreshResult)
		if txErr != nil {
			return txErr
		}
		refreshToken, txErr := tx.Token.Get(ctx, token2.WithUserGuid(id), token2.WithActive(true))
		refreshToken = &token2.Token{UserGuid: id, RefreshToken: bcryptedPass, Active: true, IP: data.IP, UserAgent: data.UserAgent}
		switch {
		case txErr == nil:
			u := token2.NewUpdate(token2.WithUpdateActive(false))
			txErr = tx.Token.Update(ctx, u)
			if txErr != nil {
				return txErr
			}
			txErr = tx.Token.Create(ctx, refreshToken)
			if txErr != nil {
				return txErr
			}
		case txErr != nil && errors.Is(txErr, sql.ErrNoRows):
			txErr = tx.Token.Create(ctx, refreshToken)
			if txErr != nil {
				return txErr
			}
		case txErr != nil:
			return txErr
		}
		refresh = refreshResult
		return nil
	})

	return token.TokensPair{AccessToken: access, RefreshToken: refresh}, nil
}

func compareUserTokenAndDBToken(userToken string, dbToken string) bool {
	shaHash := sha256.Sum256([]byte(userToken))
	err := bcrypt.CompareHashAndPassword([]byte(dbToken), shaHash[:])
	return err == nil
}

func (ts *TokenService) BlockToken(ctx context.Context, accessToken string) error {
	err := ts.uow.BlackList.Create(ctx, &blacklist.Blacklist{AccessToken: accessToken})
	if err != nil {
		return err
	}
	tokenBytes := []byte(accessToken)
	var tokenPl token.AccessTokenPayload
	_, err = jwt.Verify(tokenBytes, alg, &tokenPl)
	if err != nil {
		return err
	}
	oldToken, err := ts.uow.Token.Get(ctx, token2.WithUserGuid(tokenPl.Subject), token2.WithActive(true))
	if err != nil {
		return err
	}
	u := token2.NewUpdate(token2.WithUpdateActive(false))
	err = ts.uow.Token.Update(ctx, u, token2.WithID(oldToken.ID))
	if err != nil {
		return err
	}
	return nil
}

func (ts *TokenService) VerifyToken(ctx context.Context, accessToken string) bool {
	_, err := ts.uow.BlackList.Get(ctx, blacklist.WithAccessToken(accessToken))
	switch {
	case err != nil && errors.Is(err, sql.ErrNoRows):
		return true
	case err != nil:
		return false
	}
	return false
}

func findActiveRefreshTokenFromDBByRefreshToken(tokens token2.Tokens, userRefreshToken string) (*token2.Token, error) {
	var validRefresh *token2.Token
	for _, v := range tokens {
		if compareUserTokenAndDBToken(userRefreshToken, v.RefreshToken) {
			validRefresh = &v
			break
		}
	}
	if validRefresh == nil {
		return nil, errors.New("refresh token is not valid")
	}
	return validRefresh, nil
}
