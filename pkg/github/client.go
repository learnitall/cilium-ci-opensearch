package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	ratelimit "github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v60/github"
	"github.com/hashicorp/go-retryablehttp"
)

func GetGitHubAuthToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

func WrapWithRateLimitRetry[T any](
	ctx context.Context,
	logger *slog.Logger,
	target func() (*T, *github.Response, error),
) (*T, *github.Response, error) {
	result, resp, err := target()

	if err == nil {
		return result, resp, err
	}

	var retryAfter time.Duration
	var typ string

	if rateLimitErr, ok := err.(*github.AbuseRateLimitError); ok {
		retryAfter = *rateLimitErr.RetryAfter
		typ = "secondary"
	} else if _, ok := err.(*github.RateLimitError); ok {
		// Give an extra minute for padding
		retryAfter = time.Until(resp.Rate.Reset.Time) + time.Minute
		typ = "primary"
	} else if resp.StatusCode == 403 {
		retryAfterHeader := resp.Header.Get("X-Ratelimit-Reset")
		if retryAfterHeader == "" {
			return result, resp, err
		}

		retryAfterSec, e := strconv.ParseInt(retryAfterHeader, 10, 64)
		if e != nil {
			err = errors.Join(err, e)
			return result, resp, err
		}

		retryAfter = time.Until(time.Unix(retryAfterSec, 0))
		typ = "primary"
	} else {
		return result, resp, err
	}

	l := logger.With("err", err, "resp", resp, "type", typ, "sleepTime", retryAfter)
	l.Warn("Hit GitHub rate limit, waiting it out")

	select {
	case <-time.After(retryAfter):
		l.Info("GitHub rate limit sleep completed, retrying request")

		return WrapWithRateLimitRetry[T](ctx, logger, target)
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func NewGitHubClient(authToken string, logger *slog.Logger) (*github.Client, error) {
	// GitHub resets API requests on the top of every hour, so
	// use a two hour delay to handle waiting for the reset.
	// Two hours acts as an extra buffer.
	// We also increase the RetryMax to ensure that the exponential
	// wait gets to the hour long mark.
	// The retry client also has great logic for retrying lots of
	// different types of HTTP errors that are temporary, such as
	// 502 errors.
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 30
	retryClient.RetryWaitMax = 2 * time.Hour
	retryClient.Logger = logger.With("subsys", "github-http-client")

	// This fixes the DownloadArtifact function for the GitHub client.
	// The retryClient will automatically follow redirects, which the GitHub client
	// isn't expecting. The GitHUb client wants to received a 302 Found status code when
	// hitting the download artifact GH API endpoint, using the 'location' header in the
	// response to return the download URL for the artifact. With the retry client, a 200 Ok
	// response is given, so the GitHub client errors out with a weird
	// 'unexpected status code: 200 Ok'.
	// The ErrUseLastReponse error is a special error that signals to the HTTP client to use
	// the last repsonse it recieved, rather than the redirect.
	retryClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	rateLimiter, err := ratelimit.NewRateLimitWaiterClient(
		retryClient.StandardClient().Transport,
		ratelimit.WithLimitDetectedCallback(
			func(cc *ratelimit.CallbackContext) {
				logger.Warn(
					"Concurrent rate limit detected, sleeping",
					"sleep-until", cc.SleepUntil, "sleep-duration", cc.TotalSleepTime,
				)
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create github rate limiter: %w", err)
	}

	client := github.NewClient(rateLimiter).WithAuthToken(authToken)

	return client, nil
}
