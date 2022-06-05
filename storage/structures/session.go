package structures

import "time"

type Session struct {
	Id      string    `db:"id"`
	UserId  string    `db:"user_id"`
	Created time.Time `db:"created"`
}
