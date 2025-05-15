package main

import (
	"log/slog"
	"mergebot/config"
	_ "mergebot/handlers/gitlab"
	_ "mergebot/webhook/gitlab"
	// _ "mergebot/metrics"
)

func main() {
	var debug bool
	config.BoolVar(&debug, "debug", false, "whether debug logging is enabled (also via DEBUG)")
	config.Parse()

	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("debug is enabled")
	}

	start()
}
