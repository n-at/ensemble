package structures

type Playbook struct {
	Id          string `db:"id"`
	ProjectId   string `db:"project_id"`
	Filename    string `db:"filename"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Locked      bool   `db:"locked"`
}
