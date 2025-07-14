package token

//go:generate go tool eos generator repository --type Token --default_id=true
type Token struct {
	ID           int64  `db:"id" eos:"autoincrement"`
	UserGuid     string `db:"user_guid" eos:"user_guid"`
	RefreshToken string `db:"refresh_token" eos:"refresh_token"`
	Active       bool   `db:"active"`
	UserAgent    string
	IP           string
}
