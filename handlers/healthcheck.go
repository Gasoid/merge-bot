package handlers

import "github.com/gasoid/merge-bot/v3/logger"

func IsHealthy() bool {
	for name := range providers {
		provider, err := New(name)
		if err != nil {
			logger.Error("provider is returning error", "provider", name, "error", err)
			return false
		}

		if !provider.provider.IsHealthy() {
			return false
		}
	}

	return true
}
