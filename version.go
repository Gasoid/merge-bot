package main

import (
	"fmt"

	"github.com/gasoid/merge-bot/config"
)

var (
	Version     = "dev"
	BuildTime   = "now"
	showVersion bool
)

func init() {
	config.BoolVar(&showVersion, "version", false, "Shows version and build time")
}

func PrintVersion() {
	fmt.Printf("Version: %s\nBuildTime: %s\n", Version, BuildTime)
}
