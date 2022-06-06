package structures

import "time"

const (
	ProjectUpdateRevisionMaxLength = 200
)

type ProjectUpdate struct {
	Id        string    `db:"id"`
	ProjectId string    `db:"project_id"`
	Date      time.Time `db:"date"`
	Success   bool      `db:"success"`
	Revision  string    `db:"revision"`
	Log       string    `db:"log"`
}
