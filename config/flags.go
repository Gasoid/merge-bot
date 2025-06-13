package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3"
)

var (
	fs = flag.NewFlagSet("merge-bot", flag.ContinueOnError)
)

func StringVar(p *string, name string, value string, usage string) {
	fs.StringVar(p, name, value, usage)
}

func IntVar(p *int, name string, value int, usage string) {
	fs.IntVar(p, name, value, usage)
}

func BoolVar(p *bool, name string, value bool, usage string) {
	fs.BoolVar(p, name, value, usage)
}

func Parse() {
	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVars()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
