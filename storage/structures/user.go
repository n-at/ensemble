package structures

const (
	UserRoleAdmin    = 1
	UserRoleOperator = 2
)

type User struct {
	Id       string `db:"id"`
	Login    string `db:"login"`
	Password string `db:"password"`
	Role     int    `db:"role"`
}

func (u *User) CanViewAllProjects() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanCreateProjects() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanEditProjects() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanLockPlaybooks() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanDeletePlaybookRuns() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanControlUsers() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) CanControlKeys() bool {
	return u.Role == UserRoleAdmin
}
