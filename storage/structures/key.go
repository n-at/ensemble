package structures

type Key struct {
	Id       string `db:"id"`
	Name     string `db:"name"`
	Password string `db:"password"`
}
