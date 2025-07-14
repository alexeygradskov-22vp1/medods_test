package blacklist

//go:generate go tool eos generator repository --type Blacklist --default_id=false
type Blacklist struct {
	AccessToken string `db:"access_token"`
}
