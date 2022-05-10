package structures

import "time"

const (
	PlaybookRunModeCheck   = 1
	PlaybookRunModeExecute = 2
	PlaybookRunModeSyntax  = 3

	PlaybookRunResultRunning = 1
	PlaybookRunResultSuccess = 2
	PlaybookRunResultFailure = 3
)

type PlaybookRun struct {
	Id            string
	PlaybookId    string
	UserId        string
	Mode          int
	StartTime     time.Time
	FinishTime    time.Time
	Result        int
	InventoryFile string
	VariablesFile string
}
