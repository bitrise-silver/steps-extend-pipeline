package step_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/api"
	"github.com/bitrise-silver/steps-extend-pipeline/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExtender records Extend calls and returns configured responses.
type mockExtender struct {
	resp  api.ExtendResponse
	err   error
	calls []struct{ bitriseYML, pipelineName string }
}

func (m *mockExtender) Extend(bitriseYML, pipelineName string) (api.ExtendResponse, error) {
	m.calls = append(m.calls, struct{ bitriseYML, pipelineName string }{bitriseYML, pipelineName})
	return m.resp, m.err
}

func newStep(mock *mockExtender) step.Step {
	return step.New(
		log.NewLogger(),
		env.NewRepository(),
		func(_, _, _ string) step.PipelineExtender { return mock },
	)
}

func setRequiredEnvs(t *testing.T) {
	t.Helper()
	t.Setenv("BITRISE_APP_URL", "https://app.bitrise.io")
	t.Setenv("BITRISE_BUILD_SLUG", "test-build-slug")
	t.Setenv("BITRISE_BUILD_API_TOKEN", "test-token")
	t.Setenv("is_debug", "false")
}

// ProcessConfig tests

func TestProcessConfig_ContentInput(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "hello world")
	t.Setenv("content_file_path", "")
	t.Setenv("pipeline_name", "")

	cfg, err := newStep(nil).ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "hello world", cfg.Content)
	assert.Empty(t, cfg.PipelineName)
	assert.False(t, cfg.IsDebug)
}

func TestProcessConfig_ContentFilePath(t *testing.T) {
	setRequiredEnvs(t)
	f, err := os.CreateTemp(t.TempDir(), "content-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString("file content")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	t.Setenv("content", "")
	t.Setenv("content_file_path", f.Name())
	t.Setenv("pipeline_name", "")

	cfg, err := newStep(nil).ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "file content", cfg.Content)
}

func TestProcessConfig_PipelineName(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "yaml-content")
	t.Setenv("content_file_path", "")
	t.Setenv("pipeline_name", "my-pipeline")

	cfg, err := newStep(nil).ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "my-pipeline", cfg.PipelineName)
}

func TestProcessConfig_APICredentials(t *testing.T) {
	t.Setenv("content", "yaml")
	t.Setenv("content_file_path", "")
	t.Setenv("pipeline_name", "")
	t.Setenv("BITRISE_APP_URL", "https://custom.bitrise.io")
	t.Setenv("BITRISE_BUILD_SLUG", "abc123")
	t.Setenv("BITRISE_BUILD_API_TOKEN", "secret-token")
	t.Setenv("is_debug", "false")

	cfg, err := newStep(nil).ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "https://custom.bitrise.io", cfg.AppURL)
	assert.Equal(t, "abc123", cfg.BuildSlug)
	assert.Equal(t, "secret-token", cfg.BuildAPIToken)
}

func TestProcessConfig_IsDebug(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "msg")
	t.Setenv("content_file_path", "")
	t.Setenv("pipeline_name", "")
	t.Setenv("is_debug", "true")

	cfg, err := newStep(nil).ProcessConfig()

	require.NoError(t, err)
	assert.True(t, cfg.IsDebug)
}

func TestProcessConfig_BothSpecified_Error(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "some content")
	t.Setenv("content_file_path", "/some/path")
	t.Setenv("pipeline_name", "")

	_, err := newStep(nil).ProcessConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not both")
}

func TestProcessConfig_NeitherSpecified_Error(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "")
	t.Setenv("content_file_path", "")
	t.Setenv("pipeline_name", "")

	_, err := newStep(nil).ProcessConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")
}

func TestProcessConfig_FileNotFound_Error(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("content", "")
	t.Setenv("content_file_path", filepath.Join(t.TempDir(), "nonexistent.txt"))
	t.Setenv("pipeline_name", "")

	_, err := newStep(nil).ProcessConfig()

	require.Error(t, err)
}

// Run tests

func TestRun_CallsExtendWithContentAndPipeline(t *testing.T) {
	mock := &mockExtender{resp: api.ExtendResponse{
		Workflows: []api.Workflow{{ID: "abc", Name: "build", DependsOn: []string{"setup"}}},
	}}

	_, err := newStep(mock).Run(step.Config{
		Content:       "workflows:\n  build:\n    steps: []",
		PipelineName:  "my-pipeline",
		AppURL:        "https://app.bitrise.io",
		BuildSlug:     "slug-123",
		BuildAPIToken: "token-xyz",
	})

	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "workflows:\n  build:\n    steps: []", mock.calls[0].bitriseYML)
	assert.Equal(t, "my-pipeline", mock.calls[0].pipelineName)
}

func TestRun_ReturnsAPIError(t *testing.T) {
	mock := &mockExtender{err: errors.New("build not in a pipeline")}

	_, err := newStep(mock).Run(step.Config{
		Content:       "yaml",
		AppURL:        "https://app.bitrise.io",
		BuildSlug:     "slug",
		BuildAPIToken: "token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "build not in a pipeline")
}
