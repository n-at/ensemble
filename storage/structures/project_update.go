package structures

import "time"

type ProjectUpdate struct {
	Id        string
	ProjectId string
	Date      time.Time
	Revision  string
	Log       string
}
