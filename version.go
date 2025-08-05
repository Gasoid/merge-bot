package main

import (
	"fmt"

	"github.com/Gasoid/merge-bot/config"
)

var (
	Version     = "dev"
	BuildTime   = "now"
	showVersion bool
)

func init() {
	config.BoolVar(&showVersion, "version", false, "shows version and build time")
}

func PrintVersion() {
	fmt.Printf("Version: %s\nBuildTime: %s\n", Version, BuildTime)
}
