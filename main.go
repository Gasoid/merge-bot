package main

import (
	"github.com/Gasoid/mergebot/config"
	_ "github.com/Gasoid/mergebot/handlers/gitlab"
	"github.com/Gasoid/mergebot/logger"
	_ "github.com/Gasoid/mergebot/webhook/gitlab"
)

func main() {
	config.Parse()

	logger.New()

	start()
}
