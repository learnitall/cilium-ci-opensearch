package github

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/jstemmer/go-junit-report/v2/junit"

	"github.com/learnitall/cilium-ci-opensearch/pkg/util"
)

const PER_PAGE = 100

func GetWorkflowRuns(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	repoOwner string,
	repoName string,
	branch string,
	status string,
	event string,
	since time.Time,
	until time.Time,
	allowedJobConclucions []string,
	allowedStepConclucions []string,
	allowTestConclusions []string,
	includeTestsuites bool,
) ([]NestedWorkflowRun, error) {
	baseLogger := logger.With(
		"repoOwner", repoOwner,
		"repoName", repoName,
		"branch", branch,
		"since", since,
		"until", until,
	)

	workflowRuns := []NestedWorkflowRun{}
	runOpts := github.ListOptions{
		PerPage: PER_PAGE,
	}

	// GH API will return 1000 workflow runs at max. To ensure we don't get cut-off, pull
	// workflows one day at a time.
	currentDay := until

	for currentDay.Sub(since) >= 0 {
		// Format is based on https://docs.github.com/en/search-github/getting-started-with-searching-on-github/understanding-the-search-syntax#query-for-dates
		// GH doesn't have an "equals" date query, so we use the inclusive date range query.
		dateQuery := fmt.Sprintf("%[1]s..%[1]s", currentDay.Format("2006-01-02"))

		l := baseLogger.With("date", currentDay, "page", runOpts.Page)
		l.Info("Pulling workflow runs for repository", "event", event, "status", status)

		runs, runResp, err := WrapWithRateLimitRetry[github.WorkflowRuns](
			ctx, l,
			func() (*github.WorkflowRuns, *github.Response, error) {
				return client.Actions.ListRepositoryWorkflowRuns(
					ctx, repoOwner, repoName, &github.ListWorkflowRunsOptions{
						Event:  event,
						Branch: branch,
						// Not relevant. The Pull Requests field in the response
						// returns open PRs that have the same HEAD SHA as the returned
						// workflow run. In other words, the pull_requests field contains
						// pull requests that use the same wversion of the workflow file
						// as the returned workflow run.
						ExcludePullRequests: true,
						Status:              status,
						Created:             dateQuery,
						ListOptions:         runOpts,
					},
				)
			},
		)

		if err != nil {
			return nil, fmt.Errorf(
				"unable to pull workflow runs for repo %s/%s on branch %s: %w",
				repoOwner, repoName, branch, err,
			)
		}

		l.Info("Processing workflow runs", "total", runs.GetTotalCount(), "count", len(runs.WorkflowRuns))

		for _, runRaw := range runs.WorkflowRuns {
			run := WorkflowRun{
				ID:           runRaw.GetID(),
				Name:         runRaw.GetName(),
				NodeID:       runRaw.GetNodeID(),
				HeadBranch:   runRaw.GetHeadBranch(),
				HeadSHA:      runRaw.GetHeadSHA(),
				RunNumber:    runRaw.GetRunNumber(),
				RunAttempt:   runRaw.GetRunAttempt(),
				Event:        runRaw.GetEvent(),
				DisplayTitle: runRaw.GetDisplayTitle(),
				Status:       runRaw.GetStatus(),
				Conclusion:   runRaw.GetConclusion(),
				WorkflowID:   runRaw.GetWorkflowID(),
				URL:          runRaw.GetURL(),
				CreatedAt:    runRaw.CreatedAt.Time,
				UpdatedAt:    runRaw.UpdatedAt.Time,
				RunStartedAt: runRaw.RunStartedAt.Time,
				JobsURL:      runRaw.GetJobsURL(),
				LogsURL:      runRaw.GetLogsURL(),
				ArtifactsURL: runRaw.GetArtifactsURL(),
			}

			rawRunHeadCommit := runRaw.GetHeadCommit()
			rawRunHeadCommitAuthor := rawRunHeadCommit.GetAuthor()
			run.HeadCommit = Commit{
				Message: rawRunHeadCommit.GetMessage(),
				Author: User{
					Login: rawRunHeadCommitAuthor.GetLogin(),
					Name:  rawRunHeadCommitAuthor.GetName(),
					Email: rawRunHeadCommitAuthor.GetEmail(),
				},
				URL: rawRunHeadCommit.GetURL(),
			}

			rawRunActor := runRaw.GetActor()
			run.Actor = User{
				Login:   rawRunActor.GetLogin(),
				ID:      rawRunActor.GetID(),
				NodeID:  rawRunActor.GetNodeID(),
				Name:    rawRunActor.GetName(),
				Company: rawRunActor.GetCompany(),
				Email:   rawRunActor.GetEmail(),
			}

			rawRunTriggeringActor := runRaw.GetTriggeringActor()
			run.TriggeringActor = User{
				Login:   rawRunTriggeringActor.GetLogin(),
				ID:      rawRunTriggeringActor.GetID(),
				NodeID:  rawRunTriggeringActor.GetNodeID(),
				Name:    rawRunTriggeringActor.GetName(),
				Company: rawRunTriggeringActor.GetCompany(),
				Email:   rawRunTriggeringActor.GetEmail(),
			}

			rawRepository := runRaw.GetRepository()
			run.Repository = Repository{
				ID:       rawRepository.GetID(),
				NodeID:   rawRepository.GetNodeID(),
				Name:     rawRepository.GetName(),
				FullName: rawRepository.GetFullName(),
			}

			rawRepositoryOwner := rawRepository.GetOwner()
			run.Repository.Owner = User{
				Login:   rawRepositoryOwner.GetLogin(),
				ID:      rawRepositoryOwner.GetID(),
				NodeID:  rawRepositoryOwner.GetNodeID(),
				Name:    rawRepositoryOwner.GetName(),
				Company: rawRepositoryOwner.GetCompany(),
				Email:   rawRepositoryOwner.GetEmail(),
			}

			run.Link = fmt.Sprintf(
				"https://github.com/%s/%s/actions/runs/%d",
				run.Repository.Owner.Login, run.Repository.Name, run.ID,
			)

			nestedRun := NestedWorkflowRun{
				WorkflowRun: run,
			}

			jobs, err := GetJobsForRun(ctx, logger, client, &run, allowedJobConclucions, allowedStepConclucions)
			if err != nil {
				return nil, err
			}

			nestedRun.Jobs = jobs

			tests, err := GetTestsuitesForWorkflowRun(ctx, logger, client, &run, allowTestConclusions)
			if err != nil {
				return nil, err
			}

			if tests != nil {
				nestedRun.Tests = *tests
			}

			workflowRuns = append(workflowRuns, nestedRun)
		}

		if runResp.NextPage == 0 {
			currentDay = currentDay.Add(-(time.Hour * 24))
		}

		runOpts.Page = runResp.NextPage
	}

	return workflowRuns, nil
}

func GetTestsuitesForWorkflowRun(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	run *WorkflowRun,
	allowedTestConclusions []string,
) (*Tests, error) {
	l := logger.With("run-id", run.ID)

	l.Debug("Pulling artifacts for workflow")

	// Don't expect more than 10 artifacts per workflow run. Cilium
	// runs normally have one to two.
	artifacts, _, err := WrapWithRateLimitRetry[github.ArtifactList](
		ctx, l,
		func() (*github.ArtifactList, *github.Response, error) {
			return client.Actions.ListWorkflowRunArtifacts(
				ctx, run.Repository.Owner.Login, run.Repository.Name, run.ID,
				&github.ListOptions{
					PerPage: 10,
				},
			)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list artifacts for workflow %d: %w", run.ID, err)
	}

	l.Debug("Checking artifacts for junit file", "count", artifacts.GetTotalCount())

	var junitArtifact *github.Artifact
	for _, artifact := range artifacts.Artifacts {
		if artifact.GetName() == "cilium-junits" {
			junitArtifact = artifact
		}
	}

	if junitArtifact == nil {
		l.Debug("No junit artifact found for workflow run, ignoring")

		return nil, nil
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("cilium-junits-%d-*", run.ID))
	if err != nil {
		return nil, fmt.Errorf("unable to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFilePath)
	}()

	l.Info("Junit artifact found for workflow run, downloading", "url", junitArtifact.GetURL())

	downloadURL, downloadURLResp, err := WrapWithRateLimitRetry[url.URL](
		ctx, l,
		func() (*url.URL, *github.Response, error) {
			return client.Actions.DownloadArtifact(
				ctx, run.Repository.Owner.Login, run.Repository.Name,
				junitArtifact.GetID(), 10,
			)
		},
	)
	if err != nil {
		if downloadURLResp.StatusCode == 410 {
			l.Warn("Artiftacts for workflow run are unavailable, received status 410 Gone")

			return nil, nil
		}

		l.Debug("err", "err", err, "status", downloadURLResp.StatusCode, "status-code", downloadURLResp.StatusCode, "equal", downloadURLResp.StatusCode == 200, "body", func() string {
			b := []byte{}
			downloadURLResp.Body.Read(b)
			return string(b)
		}(), "resp", downloadURLResp.Response)

		return nil, fmt.Errorf("unable to get download url for artifact %d: %w", junitArtifact.GetID(), err)
	}

	l.Debug("Downloading cilium-junits artifact", "url", downloadURL, "dest", tmpFilePath)

	resp, err := http.Get(downloadURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to download cilium-junits artifact from %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unable to download cilium-junits artifact from %s, bad http code: %s", downloadURL, resp.Status,
		)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to write cilium-junits artifact file: %w", err)
	}

	l.Debug("Successfully downloaded cilium-junits file, reading", "path", tmpFilePath)

	zipReader, err := zip.OpenReader(tmpFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create zip reader for file %s: %w", tmpFilePath, err)
	}
	defer zipReader.Close()

	allTests := Tests{}
	for _, fil := range zipReader.File {
		if !strings.HasSuffix(fil.Name, ".xml") || fil.FileInfo().IsDir() {
			l.Debug("ignoring non-xml file in cilium-junits archive", "file", fil.Name)
			continue
		}

		l.Info("Parsing JUnit file", "name", fil.Name)

		fileReader, err := fil.Open()
		if err != nil {
			return nil, fmt.Errorf("unable to open file in cilium-junits archive for reading: %w", err)
		}
		defer fileReader.Close()

		buf := &bytes.Buffer{}

		_, err = io.Copy(buf, fileReader)
		if err != nil {
			return nil, fmt.Errorf("unable to read junit file in artifact zip: %w", err)
		}

		// Sometimes a JUnit file can be empty, so we need to rule out empty files.
		if buf.Len() == 0 {
			continue
		}

		// A JUnit file may either be:
		// 1. A junit.Testsuites object with a single junit.Testsuite.
		// 2. A single junit.Testsuite.
		// Try both options when unmarshalling.
		// Note that the XML parser thinks the Testsuites object is a valid Testsuite object, so
		// we have to try parsing into a Testsuites first.
		suite := junit.Testsuite{}
		suites := junit.Testsuites{}
		if err := xml.Unmarshal(buf.Bytes(), &suites); err != nil {
			if err2 := xml.Unmarshal(buf.Bytes(), &suite); err2 != nil {
				e := errors.Join(err, err2)
				return nil, fmt.Errorf("unable to unmarshal junit file '%s' in artifact to Testsuite or Testsuites object: %w", fil.Name, e)
			}
		} else {
			if len(suites.Suites) != 1 {
				l.Warn("Found Testsuites object with unexpected number of Testsuite objects", "file", fil.Name, "url", junitArtifact.GetURL())
				l.Debug("Found Testsuites object with unexpected number of Testsuite objects", "testsuites", suites)

				continue
			}

			suite = suites.Suites[0]
		}

		test := &Test{
			Name:          suite.Name,
			TotalTests:    suite.Tests,
			TotalFailures: suite.Failures,
			TotalErrors:   suite.Errors,
			TotalSkipped:  suite.Skipped,
		}

		if suite.Time != "" {
			duration, err := time.ParseDuration(fmt.Sprintf("%ss", suite.Time))
			if err != nil {
				return nil, fmt.Errorf("unable to parse duration '%ss': %w", suite.Time, err)
			}
			test.Duration = duration
		}

		if suite.Timestamp != "" {
			// ISO8601
			endTime, err := time.Parse("2006-01-02T15:04:05", suite.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("unable to parse timestamp '%s': %w", suite.Timestamp, err)
			}
			test.EndTime = endTime
		}

		if suite.Properties != nil {
			for _, property := range *suite.Properties {
				if property.Name == "github_job_step" {
					test.WorkflowStepName = property.Value
				}
			}
		}

		for _, testcase := range suite.Testcases {
			tc := Testcase{
				Name: testcase.Name,
			}

			// There are a couple of formats for the cilium-junits. Sometimes
			// the Status property is set, and other times it isn't. It if isn't set,
			// the status will be exposed through the different
			// result fields of the junit.Testcase.

			if testcase.Status != "" {
				tc.Status = testcase.Status
			} else {
				if testcase.Error != nil {
					tc.Status = "error"
				} else if testcase.Failure != nil {
					tc.Status = "failure"
				} else if testcase.Skipped != nil {
					tc.Status = "skipped"
				} else {
					tc.Status = "passed"
				}
			}

			if !util.Contains(allowedTestConclusions, tc.Status) {
				l.Debug(
					"Skipping test case for workflow, does not meet status criteria",
					"testcase-name", testcase.Name, "testcase-status", testcase.Status,
				)

				continue
			}

			if testcase.Time != "" {
				duration, err := time.ParseDuration(fmt.Sprintf("%ss", testcase.Time))
				if err != nil {
					return nil, fmt.Errorf("unable to parse duration '%ss': %w", testcase.Time, err)
				}
				tc.Duration = duration
			}

			test.Testcases = append(test.Testcases, tc)
		}

		allTests.TotalTests += test.TotalTests
		allTests.TotalErrors += test.TotalErrors
		allTests.TotalFailures += test.TotalFailures
		allTests.TotalSkipped += test.TotalSkipped
		allTests.Tests = append(allTests.Tests, *test)
	}

	return &allTests, nil

}

func GetLogsForJob(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	job *github.WorkflowJob,
	repoOwner string,
	repoName string,
) (string, error) {
	logger.Info("Pulling logs for job", "job-id", job.GetID())

	logURL, _, err := WrapWithRateLimitRetry[url.URL](
		ctx, logger,
		func() (*url.URL, *github.Response, error) {
			return client.Actions.GetWorkflowJobLogs(
				ctx, repoOwner, repoName, job.GetID(), 10,
			)
		},
	)
	if err != nil {
		return "", fmt.Errorf("unable to get log URL for job with ID %d: %w", job.ID, err)
	}

	resp, err := http.Get(logURL.String())
	if err != nil {
		return "", fmt.Errorf("unable to download logs for job with ID %d: %w", job.ID, err)
	}
	defer resp.Body.Close()

	buf := bytes.Buffer{}
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read logs for job with ID %d: %w", job.ID, err)
	}

	return buf.String(), nil
}

func GetJobsForRun(
	ctx context.Context,
	logger *slog.Logger,
	client *github.Client,
	run *WorkflowRun,
	allowedConclusions []string,
	allowedStepConclusions []string,
) ([]NestedJobRun, error) {
	l := logger.With("run-id", run.ID)

	l.Debug("Pulling jobs for workflow run")

	jobOpts := &github.ListOptions{
		PerPage: PER_PAGE,
	}

	jobRuns := []NestedJobRun{}

	for {
		jobs, jobResp, err := WrapWithRateLimitRetry[github.Jobs](
			ctx, l,
			func() (*github.Jobs, *github.Response, error) {
				return client.Actions.ListWorkflowJobsAttempt(
					ctx,
					run.Repository.Owner.Login,
					run.Repository.Name,
					run.ID,
					int64(run.RunAttempt),
					jobOpts,
				)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to pull jobs for workflow with ID %d: %w", run.ID, err)
		}

		l.Debug("Processing jobs runs", "total", jobs.GetTotalCount(), "count", len(jobs.Jobs))

		for _, jobRaw := range jobs.Jobs {
			if c := jobRaw.GetConclusion(); !util.Contains(allowedConclusions, c) {
				logger.Debug("Skipping job for workflow, does not meet conclusion criteria",
					"job-id", jobRaw.GetID(),
					"job-conclusion", c,
				)
				continue
			}

			job := JobRun{
				ID:          jobRaw.GetID(),
				RunID:       jobRaw.GetRunID(),
				RunURL:      jobRaw.GetRunURL(),
				NodeID:      jobRaw.GetNodeID(),
				URL:         jobRaw.GetURL(),
				Status:      jobRaw.GetStatus(),
				Conclusion:  jobRaw.GetConclusion(),
				CreatedAt:   jobRaw.GetCreatedAt().Time,
				StartedAt:   jobRaw.GetStartedAt().Time,
				CompletedAt: jobRaw.GetCompletedAt().Time,
				Name:        jobRaw.GetName(),
				Duration:    jobRaw.CompletedAt.Sub(jobRaw.StartedAt.Time),
			}
			job.Crumb = "<" + job.Name + ">" + "<" + run.Name + ">"
			job.Link = fmt.Sprintf(
				"https://github.com/%s/%s/actions/runs/%d/job/%d",
				run.Repository.Owner.Login, run.Repository.Name, run.ID, job.ID,
			)

			nestedJob := NestedJobRun{
				JobRun: job,
			}
			nestedJob.Steps = GetStepsForJob(ctx, logger, jobRaw, job.Crumb, allowedStepConclusions)

			jobRuns = append(jobRuns, nestedJob)
		}

		if jobResp.NextPage == 0 {
			break
		}

		jobOpts.Page = jobResp.NextPage
	}

	return jobRuns, nil
}

func GetStepsForJob(
	ctx context.Context,
	logger *slog.Logger,
	job *github.WorkflowJob,
	crumb string,
	allowedConclusions []string,
) []StepRun {
	steps := []StepRun{}

	for _, stepRaw := range job.Steps {
		if c := stepRaw.GetConclusion(); !util.Contains(allowedConclusions, c) {
			logger.Debug("Skipping step for job, does not meet conclusion criteria",
				"job-id", job.GetID(),
				"step-name", stepRaw.GetName(),
				"step-conclusion", c,
			)
			continue
		}

		step := StepRun{
			Name:        stepRaw.GetName(),
			Status:      stepRaw.GetStatus(),
			Conclusion:  stepRaw.GetConclusion(),
			Number:      stepRaw.GetNumber(),
			StartedAt:   stepRaw.StartedAt.Time,
			CompletedAt: stepRaw.CompletedAt.Time,
		}
		step.Crumb = "<" + step.Name + ">" + crumb

		steps = append(steps, step)
	}

	return steps
}
