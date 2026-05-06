package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/api"
	"github.com/bitrise-silver/steps-extend-pipeline/step"
)

func main() {
	logger := log.NewLogger()
	s := step.New(logger, env.NewRepository(), func(appURL, buildSlug, authToken string) step.PipelineExtender {
		return api.NewClient(appURL, buildSlug, authToken, logger)
	})

	cfg, err := s.ProcessConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1mFailed to process config: %s\x1b[0m\n", err)
		os.Exit(1)
	}

	result, err := s.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1m%s\x1b[0m\n", err)
		os.Exit(1)
	}

	if err := s.ExportOutputs(result); err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1mFailed to export outputs: %s\x1b[0m\n", err)
		os.Exit(1)
	}
}
