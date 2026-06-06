package main

import (
	"github.com/gasoid/merge-bot/v3/cache"
	"github.com/gasoid/merge-bot/v3/config"
	_ "github.com/gasoid/merge-bot/v3/handlers/gitlab"
	"github.com/gasoid/merge-bot/v3/logger"
	_ "github.com/gasoid/merge-bot/v3/webhook/gitlab"
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
