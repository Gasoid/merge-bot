package main

import (
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

	start()
}
