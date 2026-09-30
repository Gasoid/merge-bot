package main

import (
	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/gasoid/merge-bot/v3/plugins"
	_ "github.com/gasoid/merge-bot/v3/plugins/wasm"
)

func loadPlugins() {
	for plugin := range plugins.Load() {
		logger.Info("plugin loaded", "plugin name", plugin.Name)
		if plugin.Command != "" {
			handle(plugin.Command, plugin.Handler)
		}

		for _, e := range plugin.Events {
			handle(e, plugin.Handler)
		}
	}
}
