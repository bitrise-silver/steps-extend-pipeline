package step

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

// Input holds raw inputs from the Bitrise environment.
type Input struct {
	Content         string `env:"content"`
	ContentFilePath string `env:"content_file_path"`
	PipelineName    string `env:"pipeline_name"`
	IsDebug         bool   `env:"is_debug"`
}

// Config holds the validated and resolved step configuration.
type Config struct {
	Content      string
	PipelineName string
	IsDebug      bool
}

// Result holds step outputs (none for this step).
type Result struct{}

// Step implements the extend-pipeline step logic.
type Step struct {
	logger  log.Logger
	envRepo env.Repository
}

// New creates a new Step with injected dependencies.
func New(logger log.Logger, envRepo env.Repository) Step {
	return Step{
		logger:  logger,
		envRepo: envRepo,
	}
}

// ProcessConfig parses and validates step inputs.
func (s Step) ProcessConfig() (Config, error) {
	var input Input
	parser := stepconf.NewInputParser(s.envRepo)
	if err := parser.Parse(&input); err != nil {
		return Config{}, fmt.Errorf("parse inputs: %w", err)
	}
	stepconf.Print(input)

	if input.Content != "" && input.ContentFilePath != "" {
		return Config{}, fmt.Errorf("specify either 'content' or 'content_file_path', not both")
	}
	if input.Content == "" && input.ContentFilePath == "" {
		return Config{}, fmt.Errorf("either 'content' or 'content_file_path' must be specified")
	}

	content := input.Content
	if content == "" {
		data, err := os.ReadFile(input.ContentFilePath)
		if err != nil {
			return Config{}, fmt.Errorf("read content file %q: %w", input.ContentFilePath, err)
		}
		content = string(data)
	}

	return Config{
		Content:      content,
		PipelineName: input.PipelineName,
		IsDebug:      input.IsDebug,
	}, nil
}

// Run executes the main step logic.
func (s Step) Run(cfg Config) (Result, error) {
	s.logger.EnableDebugLog(cfg.IsDebug)
	s.logger.Infof("%s", cfg.Content)
	return Result{}, nil
}

// ExportOutputs exports step outputs to the Bitrise environment.
func (s Step) ExportOutputs(_ Result) error {
	return nil
}
