package structures

import "time"

const (
	PlaybookRunModeCheck   = 1
	PlaybookRunModeExecute = 2

	PlaybookRunResultRunning = 1
	PlaybookRunResultSuccess = 2
	PlaybookRunResultFailure = 3
)

type PlaybookRun struct {
	id         string
	playbookId string
	userId     string
	mode       int
	startTime  time.Time
	finishTime time.Time
	result     int
}
