package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	gh "github.com/learnitall/cilium-ci-opensearch/pkg/github"
	"github.com/learnitall/cilium-ci-opensearch/pkg/log"
	"github.com/learnitall/cilium-ci-opensearch/pkg/opensearch"
)

type typeWorkflowRunsParams struct {
	Since             time.Time
	SinceStr          string
	Until             time.Time
	UntilStr          string
	Repository        string
	Branch            string
	Events            []string
	StepConclusions   []string
	JobConclusions    []string
	TestConclusions   []string
	RunStatuses       []string
	OnlyFailedSteps   bool
	IncludeTestsuites bool
}

var (
	defaultGitHubConclusions = []string{"success", "failure", "timed_out", "cancelled"}
	defaultJUnitConclusions  = []string{"passed", "failed", "skipped"}
	workflowRunsParams       = &typeWorkflowRunsParams{}
	workflowRunsCmd          = &cobra.Command{
		Use: "runs",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			s, err := time.Parse(time.DateOnly, workflowRunsParams.SinceStr)
			if err != nil {
				return fmt.Errorf("unable to parse '%s' to format of '%s': %w", workflowRunsParams.SinceStr, time.DateOnly, err)
			}

			workflowRunsParams.Since = s

			u, err := time.Parse(time.DateOnly, workflowRunsParams.UntilStr)
			if err != nil {
				return fmt.Errorf("unable to parse '%s' to format of '%s': %w", workflowRunsParams.UntilStr, time.DateOnly, err)
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
					runs, err := gh.GetWorkflowRuns(
						ctx, logger, client,
						repoOwner, repoName, workflowRunsParams.Branch,
						status, event, workflowRunsParams.Since, workflowRunsParams.Until,
						workflowRunsParams.JobConclusions, workflowRunsParams.StepConclusions,
						workflowRunsParams.TestConclusions, workflowRunsParams.IncludeTestsuites,
					)

					if err != nil {
						logger.Error(
							"Unable to pull workflow runs",
							"event", event,
							"status", status,
							"err", err,
						)
						os.Exit(1)
					}

					for _, run := range runs {
						data, err := json.Marshal(run)
						if err != nil {
							logger.Error(
								"Unable to marshal workflow to JSON",
								"run", run,
								"err", err,
							)
							os.Exit(1)
						}

						(&opensearch.BulkEntry{
							Index: rootParams.Index,
							ID:    run.NodeID,
							Verb:  "index",
							Data:  data,
						}).Write(os.Stdout)
					}
				}
			}

		},
	}
)

func init() {
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.SinceStr, "since", "s", time.Now().Add(-time.Hour*24*7).Format(time.DateOnly),
		"Duration specifying how far back in time to query for workflow runs. "+
			"Workflows older than this time will not be returned. "+
			"Uses day granularity. Date is inclusive. Expected format is YYYY-MM-DD.",
	)
	workflowRunsCmd.PersistentFlags().StringVarP(
		&workflowRunsParams.UntilStr, "until", "u", time.Now().Format(time.DateOnly),
		"Duration specifying the latest point in time to query for workflow runs. "+
			"Workflows created after this time will not be returned. "+
			"Uses day granularity. Date is inclusive. Expected format is YYYY-MM-DD.",
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
	workflowCmd.AddCommand(workflowRunsCmd)
}
