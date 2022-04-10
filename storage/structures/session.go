package structures

import "time"

type Session struct {
	Id      string
	UserId  string
	Created time.Time
}
