package github

import (
	"time"
)

type TypeName string

const (
	TypeNameWorkflowRun = "workflow_run"
	TypeNameJobRun      = "job_run"
	TypeNameStepRun     = "step_run"
	TypeNameTestcase    = "test_case"
	TypeNameTest        = "test"
	TypeNameTestsuite   = "test_suite"
)

type User struct {
	Login   string `json:"login,omitempty"`
	ID      int64  `json:"id,omitempty"`
	NodeID  string `json:"node_id,omitempty"`
	Name    string `json:"name,omitempty"`
	Company string `json:"company,omitempty"`
	Email   string `json:"email,omitempty"`
}

type Repository struct {
	ID     int64  `json:"id,omitempty"`
	NodeID string `json:"node_id,omitempty"`
	Owner  User   `json:"owner,omitempty"`
	Name   string `json:"name,omitempty"`
	// FullName is the combined owner/name
	FullName string `json:"full_name,omitempty"`
}

type Commit struct {
	Message string `json:"message,omitempty"`
	Author  User   `json:"author,omitempty"`
	URL     string `json:"url,omitempty"`
}

type WorkflowRun struct {
	Type             string     `json:"type,omitempty"`
	ID               int64      `json:"workflow_id,omitempty"`
	Name             string     `json:"workflow_name,omitempty"`
	NodeID           string     `json:"workflow_node_id,omitempty"`
	RunNumber        int        `json:"workflow_run_number,omitempty"`
	RunAttempt       int        `json:"workflow_run_attempt,omitempty"`
	DisplayTitle     string     `json:"workflow_display_title,omitempty"`
	Status           string     `json:"workflow_status,omitempty"`
	Conclusion       string     `json:"workflow_conclusion,omitempty"`
	ParentWorkflowID int64      `json:"workflow_parent_id,omitempty"`
	URL              string     `json:"workflow_url,omitempty"`
	CreatedAt        time.Time  `json:"workflow_created_at,omitempty"`
	UpdatedAt        time.Time  `json:"workflow_updated_at,omitempty"`
	RunStartedAt     time.Time  `json:"workflow_run_started_at,omitempty"`
	JobsURL          string     `json:"workflow_jobs_url,omitempty"`
	LogsURL          string     `json:"workflow_logs_url,omitempty"`
	ArtifactsURL     string     `json:"workflow_artifacts_url,omitempty"`
	Link             string     `json:"workflow_link,omitempty"`
	Event            string     `json:"event,omitempty"`
	Actor            User       `json:"actor,omitempty"`
	TriggeringActor  User       `json:"triggering_actor,omitempty"`
	Repository       Repository `json:"repository,omitempty"`
	HeadBranch       string     `json:"head_branch,omitempty"`
	HeadSHA          string     `json:"head_sha,omitempty"`
	HeadCommit       Commit     `json:"head_commit,omitempty"`
}

type JobRun struct {
	WorkflowRun
	Type        string    `json:"type,omitempty"`
	ID          int64     `json:"job_id,omitempty"`
	RunID       int64     `json:"job_run_id,omitempty"`
	RunURL      string    `json:"job_run_url,omitempty"`
	NodeID      string    `json:"job_node_id,omitempty"`
	URL         string    `json:"job_url,omitempty"`
	Status      string    `json:"job_status,omitempty"`
	Conclusion  string    `json:"job_conclusion,omitempty"`
	CreatedAt   time.Time `json:"job_created_at,omitempty"`
	StartedAt   time.Time `json:"job_started_at,omitempty"`
	CompletedAt time.Time `json:"job_completed_at,omitempty"`
	Name        string    `json:"job_name,omitempty"`
	Logs        string    `json:"job_logs,omitempty"`
	// ErrorLogs contains log lines that contain an error.
	ErrorLogs []string `json:"job_error_logs,omitempty"`
	Link     string        `json:"job_link,omitempty"`
	Duration time.Duration `json:"job_duration,omitempty"`
}

type StepRun struct {
	JobRun
	Type        string        `json:"type,omitempty"`
	Name        string        `json:"step_name,omitempty"`
	Status      string        `json:"step_status,omitempty"`
	Conclusion  string        `json:"step_conclusion,omitempty"`
	Number      int64         `json:"step_number,omitempty"`
	StartedAt   time.Time     `json:"step_started_at,omitempty"`
	CompletedAt time.Time     `json:"step_completed_at,omitempty"`
	Duration    time.Duration `json:"step_duration,omitempty"`
}

// The following structs are based off of the structs found in
// github.com/jstemmer/go-junit-report. They are simplified to promote
// ease-of-use in OpenSearch.

type Testsuite struct {
	WorkflowRun
	Type             string        `json:"type,omitempty"`
	Name             string        `json:"test_suite_name,omitempty"`
	TotalTests       int           `json:"test_suite_total_tests,omitempty"`
	TotalFailures    int           `json:"test_suite_total_failures,omitempty"`
	TotalErrors      int           `json:"test_suite_total_errors,omitempty"`
	TotalSkipped     int           `json:"test_suite_total_skipped,omitempty"`
	Duration         time.Duration `json:"test_suite_duration,omitempty"`
	EndTime          time.Time     `json:"test_suite_end_time,omitempty"`
}

type Testcase struct {
	Testsuite
	Type      string        `json:"type,omitempty"`
	Name      string        `json:"test_case_name,omitempty"`
	Duration  time.Duration `json:"test_case_duration,omitempty"`
	Status    string        `json:"test_case_status,omitempty"`
}
