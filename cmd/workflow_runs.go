package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"

	gh "github.com/learnitall/cilium-ci-opensearch/pkg/github"
	"github.com/learnitall/cilium-ci-opensearch/pkg/log"
	"github.com/learnitall/cilium-ci-opensearch/pkg/opensearch"
)

type typeWorkflowRunsParams struct {
	Since                       time.Time
	SinceStr                    string
	Until                       time.Time
	UntilStr                    string
	Repository                  string
	Branch                      string
	Events                      []string
	StepConclusions             []string
	JobConclusions              []string
	TestConclusions             []string
	RunStatuses                 []string
	OnlyFailedSteps             bool
	IncludeTestsuites           bool
	IncludeErrorLogs            bool
	ParseWorkflowDispatchInputs bool
}

func setTestedFields(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	event,
	repoOwner,
	repoName string,
	run *gh.WorkflowRun,
	jobs *[]gh.JobRun,
) {
	if event != "workflow_dispatch" {
		run.TestedCommit = run.HeadCommit
		run.TestedSHA = run.HeadSHA
		run.TestedBranch = run.HeadBranch

		return
	}

	if !workflowRunsParams.ParseWorkflowDispatchInputs {
		return
	}

	var echoJob *gh.JobRun

	for _, job := range *jobs {
		if job.Name == "Echo Workflow Dispatch Inputs" {
			echoJob = &job
			break
		}
	}

	if echoJob == nil {
		if run.HeadBranch == "main" {
			// The echo-inputs job is currently being backported to other stable branches.
			// Do not fail if we are pulling jobs for a different branch.
			logger.Error("Got workflow_dispatch event with no echo-inputs job")
			os.Exit(1)
		}
		return
	}

	if echoJob.Conclusion != "success" {
		logger.Error("echo-inputs job was not successful")
		os.Exit(1)
	}

	logs, err := gh.GetLogsForJob(ctx, logger, client, echoJob.ID, run.Repository.Owner.Login, run.Repository.Name)
	if err != nil {
		logger.Error(
			"Unable to pull logs for echo-inputs job",
			"err", err,
		)
		os.Exit(1)
	}

	inputs := gh.ParseEchoInputsLogs(logs)

	if len(inputs) == 0 {
		logger.Error("Parsed no inputs for workflow_dispatch workflow")
		os.Exit(1)
	}

	logger.Debug(
		"Got inputs for workflow_dispatch workflow",
		"inputs", inputs,
	)
	run.WorkflowDispatchInputs = inputs

	sha, ok := inputs["SHA"]
	if !ok {
		logger.Error("Parsed inputs for workflow_dispatch workflow do not contain SHA")
		os.Exit(1)
	}

	run.TestedSHA = sha

	commit, err := gh.GetCommitBySHA(
		ctx, logger, run.Repository.Owner.Login,
		run.Repository.Name, client, sha,
	)
	if err != nil {
		logger.Error("Unable to get commit info for sha", "sha", sha)
		os.Exit(1)
	}

	run.TestedCommit = *commit

	contextRef, ok := inputs["context-ref"]
	if !ok {
		logger.Error("Parsed inputs for workflow_dispatch workflow do not contain context-ref")
		os.Exit(1)
	}

	run.TestedBranch = contextRef
}

func pullRunsWithEventAndStatus(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	repoOwner,
	repoName,
	event,
	status string,
) {
	eventLogger := logger.With(
		"event", event,
		"status", status,
	)

	runs, err := gh.GetWorkflowRuns(
		ctx, logger, client,
		repoOwner, repoName, workflowRunsParams.Branch,
		status, event, workflowRunsParams.Since, workflowRunsParams.Until,
	)
	if err != nil {
		eventLogger.Error(
			"Unable to pull workflow runs",
			"err", err,
		)
		os.Exit(1)
	}

	for _, run := range runs {
		runLogger := eventLogger.With("workflow-id", run.ID)

		jobs, steps, err := gh.GetJobsAndStepsForRun(
			ctx, logger, client, run,
			workflowRunsParams.JobConclusions,
			workflowRunsParams.StepConclusions,
			workflowRunsParams.IncludeErrorLogs,
		)
		if err != nil {
			runLogger.Error(
				"Unable to pull job and steps for workflow run",
				"err", err,
			)
			os.Exit(1)
		}

		// Fields that start with Tested* represent information regarding the tested ref.
		// These fields require special, context-aware handling.
		// TODO: Modify this function to determine if a workflow_dispatch run was scheduled by
		// ariane or executed as part of a PR. Add a flag to ignore PRs.
		// setTestedFields(ctx, runLogger, client, event, repoOwner, repoName, run, &jobs)

		if err := opensearch.BulkWriteObjects[gh.JobRun](jobs, rootParams.Index, os.Stdout); err != nil {
			runLogger.Error(
				"Unexepected error while writing job run bulk entries",
				"err", err,
			)
			os.Exit(1)
		}

		if err := opensearch.BulkWriteObjects[gh.StepRun](steps, rootParams.Index, os.Stdout); err != nil {
			runLogger.Error(
				"Unexepected error while writing step run bulk entries",
				"err", err,
			)
			os.Exit(1)
		}

		suites, cases, err := gh.GetTestsForWorkflowRun(
			ctx, logger, client, run,
			workflowRunsParams.TestConclusions,
		)
		if err != nil {
			runLogger.Error(
				"Unable to parse test cases for workflow run",
				"run", run.ID,
				"err", err,
			)
			os.Exit(1)
		}

		if err := opensearch.BulkWriteObjects[gh.Testsuite](suites, rootParams.Index, os.Stdout); err != nil {
			runLogger.Error(
				"Unexepected error while writing job run bulk entries",
				"err", err,
			)
			os.Exit(1)
		}

		if err := opensearch.BulkWriteObjects[gh.Testcase](cases, rootParams.Index, os.Stdout); err != nil {
			runLogger.Error(
				"Unexepected error while writing step run bulk entries",
				"err", err,
			)
			os.Exit(1)
		}
	}

	if err := opensearch.BulkWriteObjects[*gh.WorkflowRun](runs, rootParams.Index, os.Stdout); err != nil {
		eventLogger.Error(
			"Unexepected error while writing workflow run bulk entries",
			"err", err,
		)
		os.Exit(1)
	}
}

var (
	defaultGitHubConclusions = []string{"success", "failure", "timed_out", "cancelled", "skipped"}
	defaultJUnitConclusions  = []string{"passed", "failed", "skipped"}
	workflowRunsParams       = &typeWorkflowRunsParams{}
	workflowRunsCmd          = &cobra.Command{
		Use: "runs",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			tz := time.Now().Local().Location()

			s, err := time.ParseInLocation(timeFormatYearMonthDayHour, workflowRunsParams.SinceStr, tz)
			if err != nil {
				return fmt.Errorf("unable to parse '%s' to format of '%s': %w", workflowRunsParams.SinceStr, timeFormatYearMonthDayHour, err)
			}

			workflowRunsParams.Since = s

			u, err := time.ParseInLocation(timeFormatYearMonthDayHour, workflowRunsParams.UntilStr, tz)
			if err != nil {
				return fmt.Errorf("unable to parse '%s' to format of '%s': %w", workflowRunsParams.UntilStr, timeFormatYearMonthDayHour, err)
			}

			workflowRunsParams.Until = u

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			logger := log.NewLogger(rootParams.Verbose)

			repoParts := strings.Split(workflowRunsParams.Repository, "/")
			if len(repoParts) != 2 {
				logger.Error("Unable to extract repo owner and name from given value", "given", workflowRunsParams.Repository)
				os.Exit(1)
			}
			repoOwner := repoParts[0]
			repoName := repoParts[1]

			client, err := gh.NewGitHubClient(os.Getenv("GITHUB_TOKEN"), logger)
			if err != nil {
				logger.Error("Unable to create new GitHub Client", "err", err)
				os.Exit(1)
			}

			logger.Info(
				"Will pull workflows for the following parameters",
				"repoOwner", repoOwner,
				"repoName", repoName,
				"events", workflowRunsParams.Events,
				"runStatuses", workflowRunsParams.RunStatuses,
			)

			for _, event := range workflowRunsParams.Events {
				for _, status := range workflowRunsParams.RunStatuses {
					pullRunsWithEventAndStatus(
						ctx, logger, client, repoOwner, repoName, event, status,
					)
				}
			}

		},
	}
)

func init() {
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.SinceStr, "since", "s", time.Now().Add(-time.Hour*24*7).Format(timeFormatYearMonthDayHour),
		"Date specifying how far back in time to query for workflow runs. "+
			"Workflows older than this time will not be returned. "+
			"Uses hour granularity. Time is inclusive. Expected format is YYYY-MM-DDTHH.",
	)
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.UntilStr, "until", "u", time.Now().Format(timeFormatYearMonthDayHour),
		"Date specifying the latest point in time to query for workflow runs. "+
			"Workflows created after this time will not be returned. "+
			"Uses hour granularity. Time is inclusive. Expected format is YYYY-MM-DDTHH.",
	)
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.Repository, "repository", "r", "cilium/cilium",
		"Repository to pull workflows from in owner/name format",
	)
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.Branch, "branch", "b", "main",
		"Name of the branch to pull workflows from",
	)
	workflowRunsCmd.PersistentFlags().StringSliceVarP(
		&workflowRunsParams.Events, "events", "e", []string{"push"},
		"Only pull workflows triggered by the given events",
	)
	workflowRunsCmd.PersistentFlags().StringSliceVar(
		&workflowRunsParams.StepConclusions, "step-conclusions", defaultGitHubConclusions,
		"Only export steps with one of the given conclusions",
	)
	workflowRunsCmd.PersistentFlags().StringSliceVar(
		&workflowRunsParams.JobConclusions, "job-conclusions", defaultGitHubConclusions,
		"Only export jobs with one of the given conclusions",
	)
	workflowRunsCmd.PersistentFlags().StringSliceVar(
		&workflowRunsParams.TestConclusions, "test-conclusions", defaultJUnitConclusions,
		"Only export test cases with one of the given conclusions. Valid options are 'passed', 'skipped', 'failed'.",
	)
	workflowRunsCmd.PersistentFlags().StringSliceVar(
		&workflowRunsParams.RunStatuses, "run-statuses", defaultGitHubConclusions,
		"Only export runs with one of the given conclusions or statuses",
	)
	workflowRunsCmd.PersistentFlags().BoolVar(
		&workflowRunsParams.IncludeTestsuites, "test-suites", true,
		"Download, parse and attach JUnit artifacts if available",
	)
	workflowRunsCmd.PersistentFlags().BoolVar(
		&workflowRunsParams.IncludeErrorLogs, "error-logs", true,
		"Download logs for each job and include relevant error-specific logs",
	)
	workflowRunsCmd.PersistentFlags().BoolVar(
		&workflowRunsParams.ParseWorkflowDispatchInputs, "parse-wd-inputs", true,
		"For workflow runs triggered by workflow_dispatch that have a job named echo-inputs"+
			"parse logs to determine the inputs given to the trigger. See cilium/cilium#31424",
	)
	workflowCmd.AddCommand(workflowRunsCmd)
}
