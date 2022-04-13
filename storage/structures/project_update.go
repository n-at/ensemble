package structures

import "time"

const (
	ProjectUpdateRevisionMaxLength = 200
)

type ProjectUpdate struct {
	Id        string
	ProjectId string
	Date      time.Time
	Success   bool
	Revision  string
	Log       string
}
