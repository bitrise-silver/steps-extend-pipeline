package step

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/api"
)

// PipelineExtender is the interface for calling the Monolith extend endpoint.
type PipelineExtender interface {
	Extend(bitriseYML, pipelineName string) (api.ExtendResponse, error)
}

// Input holds raw inputs from the Bitrise environment.
type Input struct {
	Content         string `env:"content"`
	ContentFilePath string `env:"content_file_path"`
	PipelineName    string `env:"pipeline_name"`
	IsDebug         bool   `env:"is_debug"`
}

// Config holds the validated and resolved step configuration.
type Config struct {
	Content       string
	PipelineName  string
	AppURL        string
	BuildSlug     string
	BuildAPIToken string
	IsDebug       bool
}

// Result holds step outputs (none for this step).
type Result struct{}

// Step implements the extend-pipeline step logic.
type Step struct {
	logger          log.Logger
	envRepo         env.Repository
	extenderFactory func(appURL, buildSlug, authToken string) PipelineExtender
}

// New creates a new Step with injected dependencies.
// extenderFactory is called in Run with the resolved API credentials.
func New(logger log.Logger, envRepo env.Repository, extenderFactory func(appURL, buildSlug, authToken string) PipelineExtender) Step {
	return Step{
		logger:          logger,
		envRepo:         envRepo,
		extenderFactory: extenderFactory,
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

	appURL := s.envRepo.Get("BITRISE_APP_URL")
	if appURL == "" {
		return Config{}, fmt.Errorf("BITRISE_APP_URL is not set")
	}
	buildSlug := s.envRepo.Get("BITRISE_BUILD_SLUG")
	if buildSlug == "" {
		return Config{}, fmt.Errorf("BITRISE_BUILD_SLUG is not set")
	}
	buildAPIToken := s.envRepo.Get("BITRISE_BUILD_API_TOKEN")
	if buildAPIToken == "" {
		return Config{}, fmt.Errorf("BITRISE_BUILD_API_TOKEN is not set")
	}

	return Config{
		Content:       content,
		PipelineName:  input.PipelineName,
		AppURL:        appURL,
		BuildSlug:     buildSlug,
		BuildAPIToken: buildAPIToken,
		IsDebug:       input.IsDebug,
	}, nil
}

// Run calls the Monolith extend endpoint and logs the resulting workflows.
func (s Step) Run(cfg Config) (Result, error) {
	s.logger.EnableDebugLog(cfg.IsDebug)

	extender := s.extenderFactory(cfg.AppURL, cfg.BuildSlug, cfg.BuildAPIToken)
	resp, err := extender.Extend(cfg.Content, cfg.PipelineName)
	if err != nil {
		return Result{}, err
	}

	s.logger.Donef("Pipeline extended with %d workflow(s):", len(resp.Workflows))
	for _, wf := range resp.Workflows {
		if len(wf.DependsOn) > 0 {
			s.logger.Infof("  - %s (depends on: %s)", wf.Name, strings.Join(wf.DependsOn, ", "))
		} else {
			s.logger.Infof("  - %s", wf.Name)
		}
	}

	return Result{}, nil
}

// ExportOutputs exports step outputs to the Bitrise environment.
func (s Step) ExportOutputs(_ Result) error {
	return nil
}
