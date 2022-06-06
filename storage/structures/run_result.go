package structures

type RunResult struct {
	Id     string `db:"id"`
	RunId  string `db:"run_id"`
	Output string `db:"output"`
	Error  string `db:"error"`
}
