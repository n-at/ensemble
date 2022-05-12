package structures

import (
	"encoding/json"
	"time"
)

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
	Destination string              `json:"dest"`
	Diff        AnsibleResultDiff   `json:"diff"`
	Facts       AnsibleFacts        `json:"ansible_facts"`
	ReturnCode  int                 `json:"rc"`
	Message     string              `json:"msg"`
	Stdout      string              `json:"stdout"`
	Stderr      string              `json:"stderr"`
	ItemResults []AnsibleItemResult `json:"results"`
}

type AnsibleItemResult struct {
	Item        string            `json:"item"`
	Changed     bool              `json:"changed"`
	Failed      bool              `json:"failed"`
	Skipped     bool              `json:"skipped"`
	Destination string            `json:"dest"`
	Diff        AnsibleResultDiff `json:"diff"`
	ReturnCode  int               `json:"rc"`
	Message     string            `json:"msg"`
	Stdout      string            `json:"stdout"`
	Stderr      string            `json:"stderr"`
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

type AnsibleResultDiff struct {
	Items        []AnsibleCheckDiff
	TemplateDiff AnsibleTemplateDiff
}

//AnsibleCheckDiff Diff from check playbook execution
type AnsibleCheckDiff struct {
	Before       string `json:"before"`
	BeforeHeader string `json:"before_header"`
	After        string `json:"after"`
	AfterHeader  string `json:"after_header"`
}

//AnsibleTemplateDiff Diff from template task execution
type AnsibleTemplateDiff struct {
	Before AnsibleTemplateDiffPath `json:"before"`
	After  AnsibleTemplateDiffPath `json:"after"`
}

type AnsibleTemplateDiffPath struct {
	Path string `json:"path"`
}

///////////////////////////////////////////////////////////////////////////////

type AnsibleFacts struct {
	Architecture             string `json:"ansible_architecture"`
	HostName                 string `json:"ansible_hostname"`
	System                   string `json:"ansible_system"`
	BiosDate                 string `json:"ansible_bios_date"`
	BiosVendor               string `json:"ansible_bios_vendor"`
	BiosVersion              string `json:"ansible_bios_version"`
	BoardVendor              string `json:"ansible_board_vendor"`
	BoardName                string `json:"ansible_board_name"`
	BoardVersion             string `json:"ansible_board_version"`
	Distribution             string `json:"ansible_distribution"`
	DistributionFile         bool   `json:"ansible_distribution_file_parsed"`
	DistributionFileVariety  string `json:"ansible_distribution_file_variety"`
	DistributionVersionMajor string `json:"ansible_distribution_major_version"`
	DistributionVersion      string `json:"ansible_distribution_version"`
	DistributionRelease      string `json:"ansible_distribution_release"`
	Kernel                   string `json:"ansible_kernel"`
	KernelVersion            string `json:"ansible_kernel_version"`
	OsFamily                 string `json:"ansible_os_family"`
	OsPackageManager         string `json:"ansible_pkg_mgr"`
	OsServiceManager         string `json:"ansible_service_mgr"`
	MemoryTotal              int    `json:"ansible_memtotal_mb"`
	MemorySwapTotal          int    `json:"ansible_swaptotal_mb"`
	CpuCount                 int    `json:"ansible_processor_count"`
	CpuCores                 int    `json:"ansible_processor_cores"`
	PythonVersion            string `json:"ansible_python_version"`
}

///////////////////////////////////////////////////////////////////////////////

func (d AnsibleDuration) StartTime() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", d.Start)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (d AnsibleDuration) EndTime() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", d.End)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (d AnsibleDuration) RunTime() time.Duration {
	start := d.StartTime()
	end := d.EndTime()
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start)
}

func (d *AnsibleResultDiff) UnmarshalJSON(data []byte) error {
	var diff []AnsibleCheckDiff
	err := json.Unmarshal(data, &diff)
	if err != nil {
		var diff AnsibleTemplateDiff
		err = json.Unmarshal(data, &diff)
		if err == nil {
			d.TemplateDiff = diff
		}
	} else {
		d.Items = diff
	}
	return nil
}
