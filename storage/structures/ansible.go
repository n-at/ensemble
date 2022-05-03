package structures

import "time"

type AnsibleExecution struct {
	Stats map[string]AnsibleStats `json:"stats"`
	Plays []AnsiblePlay           `json:"plays"`
}

type AnsiblePlay struct {
	PlayInfo AnsiblePlayInfo `json:"play"`
	Tasks    []AnsibleTask   `json:"tasks"`
}

type AnsibleTask struct {
	TaskInfo    AnsibleTaskInfo              `json:"task"`
	TaskResults map[string]AnsibleTaskResult `json:"hosts"`
}

type AnsiblePlayInfo struct {
	Duration AnsibleDuration `json:"duration"`
	Id       string          `json:"id"`
	Name     string          `json:"name"`
}

type AnsibleTaskInfo struct {
	Duration AnsibleDuration `json:"duration"`
	Id       string          `json:"id"`
	Name     string          `json:"name"`
}

type AnsibleDuration struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type AnsibleTaskResult struct {
	Action      string              `json:"action"`
	Changed     bool                `json:"changed"`
	Failed      bool                `json:"failed"`
	Skipped     bool                `json:"skipped"`
	Diff        []AnsibleDiff       `json:"diff"`
	ReturnCode  bool                `json:"rc"`
	Message     string              `json:"msg"`
	Stdout      []string            `json:"stdout_lines"`
	Stderr      []string            `json:"stderr_lines"`
	ItemResults []AnsibleItemResult `json:"results"`
}

type AnsibleDiff struct {
	Before       string `json:"before"`
	BeforeHeader string `json:"before_header"`
	After        string `json:"after"`
	AfterHeader  string `json:"after_header"`
}

type AnsibleItemResult struct {
	Item       string        `json:"item"`
	Changed    bool          `json:"changed"`
	Failed     bool          `json:"failed"`
	Skipped    bool          `json:"skipped"`
	Diff       []AnsibleDiff `json:"diff"`
	ReturnCode bool          `json:"rc"`
	Message    string        `json:"msg"`
	Stdout     []string      `json:"stdout"`
	Stderr     []string      `json:"stderr"`
}

type AnsibleStats struct {
	Ok          int `json:"ok"`
	Changed     int `json:"changed"`
	Ignored     int `json:"ignored"`
	Skipped     int `json:"skipped"`
	Rescued     int `json:"rescued"`
	Failures    int `json:"failures"`
	Unreachable int `json:"unreachable"`
}

///////////////////////////////////////////////////////////////////////////////

func (d AnsibleDuration) StartDate() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", d.Start)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (d AnsibleDuration) EndDate() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", d.End)
	if err != nil {
		return time.Time{}
	}
	return t
}
