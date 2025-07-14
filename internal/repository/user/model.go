package user

//go:generate go tool eos generator repository --type User --default_id=false
type User struct {
	Guid string `db:"guid" eos:"guid"`
}
