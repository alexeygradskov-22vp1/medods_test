package token

import "github.com/gbrlsnchs/jwt/v3"

type AccessTokenPayload struct {
	jwt.Payload
	Key string `json:"Key"`
}

type TokensPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenPayload struct {
	Time string
	Key  string
}
