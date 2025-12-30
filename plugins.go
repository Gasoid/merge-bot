package main

import (
	"github.com/gasoid/merge-bot/logger"
	"github.com/gasoid/merge-bot/plugins"
)

func loadPlugins() {
	for plugin := range plugins.Load() {
		logger.Info("plugin loaded", "plugin name", plugin.Name)
		handle(plugin.Command, plugin.Handler)
	}
}
