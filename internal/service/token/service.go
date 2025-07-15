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
	"time"
)

const (
	Issuer string = "medods_test"
	JwtId  string = "JWTID"
)

type TokenService struct {
	c      *external.ExternalServiceClient
	blRepo *blacklist.BlacklistRepository
	tRepo  *token2.TokenRepository
}

func NewTService(c *external.ExternalServiceClient, blRepo *blacklist.BlacklistRepository, tRepo *token2.TokenRepository) *TokenService {
	return &TokenService{c: c, blRepo: blRepo, tRepo: tRepo}
}

var alg = jwt.NewHS512([]byte("secret_phrase"))

// build access token
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
	//build jwt token by alg SHA512
	buildedToken, err := jwt.Sign(pl, alg)
	if err != nil {
		return "", fmt.Errorf("jwt sign err %w", err)
	}

	return string(buildedToken), nil
}

// build refresh token
func buildRefreshToken(key string) (refresh string, err error) {
	//build payload
	pl := token.RefreshTokenPayload{Time: time.Now().String(), Key: key}

	//marshal payload to bytes
	plBytes, err := json.Marshal(pl)
	if err != nil {
		return "", err
	}

	//calc len of slice base64 encoded payload
	encodedLen := base64.RawURLEncoding.EncodedLen(len(plBytes))

	refreshBytes := make([]byte, encodedLen)

	// encode payload to base64
	base64.RawURLEncoding.Encode(refreshBytes, plBytes)

	refresh = string(refreshBytes)
	return
}

func (ts *TokenService) Refresh(ctx context.Context, tokenPayload *token.TokensPair, data request.RequestData) (err error) {
	//convert access token to bytes and verify it
	tokenBytes := []byte(tokenPayload.AccessToken)
	var tokenPl token.AccessTokenPayload
	_, err = jwt.Verify(tokenBytes, alg, &tokenPl)
	if err != nil {
		return fmt.Errorf("verify token error: %w", err)
	}

	//decode refresh token from base64 string to payload struct
	var refreshTokenPl token.RefreshTokenPayload
	err = decodeRefreshToken(&refreshTokenPl, tokenPayload.RefreshToken)
	if err != nil {
		return fmt.Errorf("decode refresh token pl error: %w", err)
	}

	// compare keys of refresh and access tokens
	if refreshTokenPl.Key != tokenPl.Key {
		return fmt.Errorf("invalid token")
	}

	var access, refresh string

	// get old active refresh tokens from database by user guid
	oldTokens, err := ts.tRepo.GetManyBy(ctx, []token2.FieldName{token2.UserGuidField, token2.ActiveField}, []interface{}{tokenPl.Subject, true})
	if err != nil {
		return fmt.Errorf("get old active tokens from db error: %w", err)
	}

	// generate uuid for unique key of payload
	genUuid, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("generate uuid for payload error: %w", err)
	}

	// validate refresh token from user
	validRefresh, err := findActiveRefreshTokenFromDBByRefreshToken(oldTokens, tokenPayload.RefreshToken)
	if err != nil {
		return fmt.Errorf("validate users refresh token error: %w", err)
	}

	//compare user-agent
	if validRefresh.UserAgent != data.UserAgent {
		err = ts.BlockToken(ctx, tokenPayload.AccessToken)
		return fmt.Errorf("user agent mismatch")
	}

	//compare ip
	if validRefresh.IP != data.IP {

		//send webhook about changes of ip address
		err = ts.c.SendWebhook("dummy_url", map[string]string{"warn": fmt.Sprintf("changed id from %s to %s", validRefresh.IP, data.IP)})
		if err != nil {
			log.Printf("error while sending webhook: %s", err)
		}
	}

	//build access token
	access, err = buildAccessToken(validRefresh.UserGuid, genUuid.String())
	if err != nil {
		return fmt.Errorf("build access token error: %w", err)
	}

	//build refresh token
	refresh, err = buildRefreshToken(genUuid.String())
	if err != nil {
		return fmt.Errorf("build refresh token: %w", err)
	}

	//process refresh token for save to database
	bcrypted, txErr := processForStorageInDatabase(refresh)
	if txErr != nil {
		return fmt.Errorf("proccess token to database format error: %w", txErr)
	}

	// deactivate old refresh token
	validRefresh.Active = false
	err = ts.tRepo.Update(ctx, validRefresh, []token2.FieldName{token2.IdField}, []interface{}{validRefresh.ID})
	if err != nil {
		return fmt.Errorf("deactivate old refresh token error: %w", txErr)
	}

	// save new refresh token to database
	created := &token2.Token{RefreshToken: bcrypted, UserGuid: validRefresh.UserGuid, Active: true, UserAgent: data.UserAgent, IP: data.IP}
	err = ts.tRepo.Create(ctx, created)
	if err != nil {
		return fmt.Errorf("save new refresh token error: %w", txErr)
	}

	tokenPayload.AccessToken = access
	tokenPayload.RefreshToken = refresh
	return err
}

// convert token to sha256 and crypt by bcrypt alg
func processForStorageInDatabase(token string) (string, error) {
	refreshBytes := []byte(token)
	sh := sha256.Sum256(refreshBytes)
	hashed, err := bcrypt.GenerateFromPassword(sh[:], bcrypt.DefaultCost)
	return string(hashed), err
}

// Valid validate access token
func (ts *TokenService) Valid(accessToken string, dst *token.AccessTokenPayload) error {
	// try to verify jwt token
	tokenBytes := []byte(accessToken)
	_, err := jwt.Verify(tokenBytes, alg, &dst)
	if err != nil {
		return err
	}

	//check of expiration time
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

	//generate new uuid for payload
	genUuid, err := uuid.NewUUID()
	if err != nil {
		return token.TokensPair{}, fmt.Errorf("generate uuid for token payload: %w", err)
	}

	//build access token
	access, err := buildAccessToken(id, genUuid.String())
	if err != nil {
		return token.TokensPair{}, fmt.Errorf("build access token error: %w", err)
	}

	//build refresh token
	refreshResult, err := buildRefreshToken(genUuid.String())
	if err != nil {
		return token.TokensPair{}, fmt.Errorf("build refresh token error: %w", err)
	}

	//process refresh token to database format
	bcryptedPass, err := processForStorageInDatabase(refreshResult)
	if err != nil {
		return token.TokensPair{}, fmt.Errorf("proccess token to database format error: %w", err)
	}

	//find exists active tokens
	oldToken, err := ts.tRepo.GetOneBy(ctx, []token2.FieldName{token2.UserGuidField, token2.ActiveField}, []interface{}{id, true})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

		} else {
			return token.TokensPair{}, fmt.Errorf("get old token: %w", err)
		}
	} else {
		oldToken.Active = false
		if err = ts.tRepo.Update(ctx, oldToken, []token2.FieldName{token2.IdField}, []interface{}{oldToken.ID}); err != nil {
			return token.TokensPair{}, fmt.Errorf("deactivate old token: %w", err)
		}
	}

	//create new refresh token
	refreshToken := &token2.Token{UserGuid: id, RefreshToken: bcryptedPass, Active: true, IP: data.IP, UserAgent: data.UserAgent}

	err = ts.tRepo.Create(ctx, refreshToken) //create new token
	if err != nil {
		return token.TokensPair{}, err
	}

	refresh = refreshResult

	return token.TokensPair{AccessToken: access, RefreshToken: refresh}, nil
}

func compareUserTokenAndDBToken(userToken string, dbToken string) bool {
	shaHash := sha256.Sum256([]byte(userToken))
	err := bcrypt.CompareHashAndPassword([]byte(dbToken), shaHash[:])
	return err == nil
}

func (ts *TokenService) BlockToken(ctx context.Context, accessToken string) error {
	//block token
	err := ts.blRepo.Create(ctx, &blacklist.Blacklist{AccessToken: accessToken})
	if err != nil {
		return fmt.Errorf("block token: %w", err)
	}

	//verify access token and store to struct
	tokenBytes := []byte(accessToken)
	var tokenPl token.AccessTokenPayload
	_, err = jwt.Verify(tokenBytes, alg, &tokenPl)
	if err != nil {
		return fmt.Errorf("verify token: %w", err)
	}

	//find old refresh token
	oldToken, err := ts.tRepo.GetOneBy(ctx,
		[]token2.FieldName{token2.UserGuidField, token2.ActiveField},
		[]interface{}{tokenPl.Subject, true})
	if err != nil {
		return fmt.Errorf("get old token: %w", err)
	}

	//deactivate old refresh token
	oldToken.Active = false
	err = ts.tRepo.Update(ctx, oldToken, []token2.FieldName{token2.IdField}, []interface{}{oldToken.ID})
	if err != nil {
		return fmt.Errorf("deactivate token: %w", err)
	}
	return nil
}

func (ts *TokenService) VerifyToken(ctx context.Context, accessToken string) bool {
	// find token in black list
	_, err := ts.blRepo.GetOneBy(ctx, []blacklist.FieldName{blacklist.AccessTokenFieldName}, []interface{}{accessToken})

	//handle error
	switch {
	case err != nil && errors.Is(err, sql.ErrNoRows):
		return true // token not found
	case err != nil:
		return false // error
	}
	return false
}

func findActiveRefreshTokenFromDBByRefreshToken(tokens []token2.Token, userRefreshToken string) (*token2.Token, error) {
	var validRefresh *token2.Token
	//compare bcrypt user token and bcrypt tokens from database
	for _, v := range tokens {
		if compareUserTokenAndDBToken(userRefreshToken, v.RefreshToken) {
			validRefresh = &v
			break
		}
	}
	//if token not found in database then it not valid
	if validRefresh == nil {
		return nil, errors.New("refresh token is not valid")
	}
	return validRefresh, nil
}
