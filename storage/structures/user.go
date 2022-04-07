package structures

const (
	UserRoleAdmin    = 1
	UserRoleOperator = 2
)

type User struct {
	id       string
	login    string
	password string
	role     int
}
