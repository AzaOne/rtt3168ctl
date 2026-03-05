package main

import (
	"os"

	"rtt3168ctl/internal/core/config"
	"rtt3168ctl/internal/core/kernel"
	"rtt3168ctl/internal/core/logging"
	"rtt3168ctl/internal/facade"
	"rtt3168ctl/internal/interfaces/cli"
)

func main() {
	logger := logging.New(os.Stderr)

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatalf("configuration error: %v", err)
	}

	cmd, shouldExit, err := cli.Parse(os.Args[1:], os.Args[0], os.Stderr)
	if err != nil {
		logger.Fatalf("CLI parse error: %v", err)
	}
	if shouldExit {
		return
	}

	app := facade.New(kernel.New(cfg, logger))
	if err := app.Execute(cmd, os.Stdout); err != nil {
		logger.Fatalf("execution error: %v", err)
	}
}
