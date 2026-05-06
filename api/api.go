package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
)

// Client handles HTTP communication with the Bitrise Monolith extend endpoint.
type Client struct {
	httpClient *http.Client
	url        string
	authToken  string
}

// NewClient creates a Client for the given build.
// Retry policy: 5xx and transport errors use library defaults; 409 Conflict is
// retried up to 3 times with a fixed 500 ms delay between attempts.
func NewClient(appURL, buildSlug, authToken string, logger log.Logger) Client {
	rc := retryhttp.NewClient(logger)
	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 500 * time.Millisecond
	rc.CheckRetry = retryOnConflict

	return Client{
		httpClient: rc.StandardClient(),
		url:        fmt.Sprintf("%s/pipeline/workflow_builds/%s/extend", appURL, buildSlug),
		authToken:  authToken,
	}
}

// retryOnConflict extends the default retry policy to also retry HTTP 409 responses.
func retryOnConflict(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if resp != nil && resp.StatusCode == http.StatusConflict {
		return true, nil
	}
	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// ExtendRequest is the JSON body sent to the extend endpoint.
type ExtendRequest struct {
	BitriseYML   string `json:"bitrise_yml"`
	PipelineName string `json:"pipeline_name,omitempty"`
}

// Workflow is a single workflow entry in the extend response.
type Workflow struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	DependsOn []string `json:"depends_on"`
}

// ExtendResponse is the success body returned by the extend endpoint.
type ExtendResponse struct {
	Workflows []Workflow `json:"workflows"`
}

// Extend POSTs to the Monolith extend endpoint and returns the created workflows.
// pipeline_name is omitted from the request body when empty.
func (c Client) Extend(bitriseYML, pipelineName string) (ExtendResponse, error) {
	body, err := json.Marshal(ExtendRequest{BitriseYML: bitriseYML, PipelineName: pipelineName})
	if err != nil {
		return ExtendResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return ExtendResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-HTTP-BUILD-API-TOKEN", c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ExtendResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	return parseResponse(resp)
}

func parseResponse(resp *http.Response) (ExtendResponse, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExtendResponse{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result ExtendResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return ExtendResponse{}, fmt.Errorf("parse response: %w", err)
		}
		return result, nil
	}

	// Surface the user-facing error_msg directly — pipeline-service errors are user-facing.
	var errBody struct {
		ErrorMsg string `json:"error_msg"`
	}
	if err := json.Unmarshal(respBody, &errBody); err == nil && errBody.ErrorMsg != "" {
		return ExtendResponse{}, errors.New(errBody.ErrorMsg)
	}
	return ExtendResponse{}, fmt.Errorf("request failed (HTTP %d): %s", resp.StatusCode, string(respBody))
}
