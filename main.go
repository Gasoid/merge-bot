package main

import (
	"github.com/gasoid/merge-bot/cache"
	"github.com/gasoid/merge-bot/config"
	_ "github.com/gasoid/merge-bot/handlers/gitlab"
	"github.com/gasoid/merge-bot/logger"
	_ "github.com/gasoid/merge-bot/webhook/gitlab"
)

func main() {
	config.Parse()

	if showVersion {
		PrintVersion()
		return
	}

	logger.New()

	if err := cache.Init(); err != nil {
		logger.Error("cache can't be initialized", "error", err)
	}

	startMetricsEndpoint()
	start()
}
