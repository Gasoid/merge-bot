package main

import (
	"github.com/Gasoid/merge-bot/config"
	_ "github.com/Gasoid/merge-bot/handlers/gitlab"
	"github.com/Gasoid/merge-bot/logger"
	_ "github.com/Gasoid/merge-bot/webhook/gitlab"
)

func main() {
	config.Parse()

	logger.New()

	if showVersion {
		PrintVersion()
		return
	}

	start()
}
