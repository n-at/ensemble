package structures

const (
	UserRoleAdmin    = 1
	UserRoleOperator = 2
)

type User struct {
	Id       string
	Login    string
	Password string
	Role     int
}
