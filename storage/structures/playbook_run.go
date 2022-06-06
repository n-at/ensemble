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
	Id            string    `db:"id"`
	PlaybookId    string    `db:"playbook_id"`
	UserId        string    `db:"user_id"`
	Mode          int       `db:"mode"`
	StartTime     time.Time `db:"start_time"`
	FinishTime    time.Time `db:"finish_time"`
	Result        int       `db:"result"`
	InventoryFile string    `db:"inventory_file"`
	VariablesFile string    `db:"variables_file"`
}

func (r *PlaybookRun) RunTime() time.Duration {
	if r.StartTime.IsZero() || r.FinishTime.IsZero() {
		return 0
	}
	return r.FinishTime.Sub(r.StartTime)
}
