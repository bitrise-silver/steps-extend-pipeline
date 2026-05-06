package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/step"
)

func main() {
	logger := log.NewLogger()
	envRepo := env.NewRepository()
	s := step.New(logger, envRepo)

	cfg, err := s.ProcessConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1mFailed to process config: %s\x1b[0m\n", err)
		os.Exit(1)
	}

	result, err := s.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1mFailed to run step: %s\x1b[0m\n", err)
		os.Exit(1)
	}

	if err := s.ExportOutputs(result); err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31;1mFailed to export outputs: %s\x1b[0m\n", err)
		os.Exit(1)
	}
}
