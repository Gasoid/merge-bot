package main

import (
	"mergebot/config"
	_ "mergebot/handlers/gitlab"
	"mergebot/logger"
	_ "mergebot/webhook/gitlab"
)

func main() {
	config.Parse()

	logger.New()

	start()
}
