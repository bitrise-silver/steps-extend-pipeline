package step_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStep() step.Step {
	return step.New(log.NewLogger(), env.NewRepository())
}

func TestProcessConfig_ContentInput(t *testing.T) {
	t.Setenv("content", "hello world")
	t.Setenv("content_file_path", "")
	t.Setenv("is_debug", "false")

	cfg, err := newStep().ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "hello world", cfg.Content)
	assert.False(t, cfg.IsDebug)
}

func TestProcessConfig_ContentFilePath(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "content-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString("file content")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	t.Setenv("content", "")
	t.Setenv("content_file_path", f.Name())
	t.Setenv("is_debug", "false")

	cfg, err := newStep().ProcessConfig()

	require.NoError(t, err)
	assert.Equal(t, "file content", cfg.Content)
}

func TestProcessConfig_IsDebug(t *testing.T) {
	t.Setenv("content", "msg")
	t.Setenv("content_file_path", "")
	t.Setenv("is_debug", "true")

	cfg, err := newStep().ProcessConfig()

	require.NoError(t, err)
	assert.True(t, cfg.IsDebug)
}

func TestProcessConfig_BothSpecified_Error(t *testing.T) {
	t.Setenv("content", "some content")
	t.Setenv("content_file_path", "/some/path")
	t.Setenv("is_debug", "false")

	_, err := newStep().ProcessConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not both")
}

func TestProcessConfig_NeitherSpecified_Error(t *testing.T) {
	t.Setenv("content", "")
	t.Setenv("content_file_path", "")
	t.Setenv("is_debug", "false")

	_, err := newStep().ProcessConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")
}

func TestProcessConfig_FileNotFound_Error(t *testing.T) {
	t.Setenv("content", "")
	t.Setenv("content_file_path", filepath.Join(t.TempDir(), "nonexistent.txt"))
	t.Setenv("is_debug", "false")

	_, err := newStep().ProcessConfig()

	require.Error(t, err)
}

func TestRun_LogsContent(t *testing.T) {
	s := step.New(log.NewLogger(), env.NewRepository())

	_, err := s.Run(step.Config{Content: "test output", IsDebug: false})

	require.NoError(t, err)
}
