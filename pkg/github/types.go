package github

import (
	"time"
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
	ID              int64      `json:"id,omitempty"`
	Name            string     `json:"name,omitempty"`
	NodeID          string     `json:"node_id,omitempty"`
	HeadBranch      string     `json:"head_branch,omitempty"`
	HeadSHA         string     `json:"head_sha,omitempty"`
	HeadCommit      Commit     `json:"head_commit,omitempty"`
	RunNumber       int        `json:"run_number,omitempty"`
	RunAttempt      int        `json:"run_attempt,omitempty"`
	Event           string     `json:"event,omitempty"`
	DisplayTitle    string     `json:"display_title,omitempty"`
	Status          string     `json:"status,omitempty"`
	Conclusion      string     `json:"conclusion,omitempty"`
	WorkflowID      int64      `json:"workflow_id,omitempty"`
	URL             string     `json:"url,omitempty"`
	CreatedAt       time.Time  `json:"created_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at,omitempty"`
	RunStartedAt    time.Time  `json:"run_started_at,omitempty"`
	JobsURL         string     `json:"jobs_url,omitempty"`
	LogsURL         string     `json:"logs_url,omitempty"`
	ArtifactsURL    string     `json:"artifacts_url,omitempty"`
	Actor           User       `json:"actor,omitempty"`
	TriggeringActor User       `json:"triggering_actor,omitempty"`
	Repository      Repository `json:"repository,omitempty"`
	Link            string     `json:"link,omitempty"`
}

type NestedWorkflowRun struct {
	WorkflowRun
	Jobs  []NestedJobRun `json:"jobs,omitempty"`
	Tests Tests          `json:"tests,omitempty"`
}

type JobRun struct {
	ID          int64     `json:"id,omitempty"`
	RunID       int64     `json:"run_id,omitempty"`
	RunURL      string    `json:"run_url,omitempty"`
	NodeID      string    `json:"node_id,omitempty"`
	URL         string    `json:"url,omitempty"`
	Status      string    `json:"status,omitempty"`
	Conclusion  string    `json:"conclusion,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Name        string    `json:"name,omitempty"`
	Logs        string    `json:"logs,omitempty"`
	// Crumb represents the location the job is found:
	// workflow_name/job_name
	Crumb    string        `json:"crumb,omitempty"`
	Link     string        `json:"link,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

type NestedJobRun struct {
	JobRun
	Steps []StepRun `json:"steps,omitempty"`
}

type StepRun struct {
	Name        string    `json:"name,omitempty"`
	Status      string    `json:"status,omitempty"`
	Conclusion  string    `json:"conclusion,omitempty"`
	Number      int64     `json:"number,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	// Crump represents the location the job is found:
	// workflow_naem/job_name/step_name
	Crumb string `json:"crumb,omitempty"`
}

// The following structs are based off of the structs found in
// github.com/jstemmer/go-junit-report. They are simplified to promote
// ease-of-use in OpenSearch.

type Testcase struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Status   string        `json:"status"`
}

type Test struct {
	Name             string        `json:"name"`
	TotalTests       int           `json:"totalTests"`
	TotalFailures    int           `json:"totalFailures"`
	TotalErrors      int           `json:"totalErrors"`
	TotalSkipped     int           `json:"totalSkipped"`
	Duration         time.Duration `json:"duration"`
	EndTime          time.Time     `json:"endTime"`
	WorkflowStepName string        `json:"workflowStepName"`

	Testcases []Testcase `json:"testCases,omitempty"`
}

type Tests struct {
	Duration      time.Duration `json:"duration"`
	TotalTests    int           `json:"totalTests"`
	TotalErrors   int           `json:"totalErrors"`
	TotalFailures int           `json:"totalFailures"`
	TotalSkipped  int           `json:"totalSkipped"`
	Tests         []Test        `json:"tests,omitempty"`
}
