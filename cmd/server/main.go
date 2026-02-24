package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/canonical/ubuntu-manpages-operator/internal/config"
	"github.com/canonical/ubuntu-manpages-operator/internal/logging"
	"github.com/canonical/ubuntu-manpages-operator/internal/web"
)

func main() {
	configPath := flag.String("config", config.DefaultPath(), "Path to config JSON")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	addr := flag.String("addr", ":8080", "HTTP bind address")
	flag.Parse()

	logger := logging.BuildLogger(*logLevel)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	server := web.NewServer(cfg, logger)
	if err := server.ListenAndServe(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
