package structures

import "time"

type ProjectUpdate struct {
	id        string
	projectId string
	date      time.Time
	revision  string
	log       string
}
