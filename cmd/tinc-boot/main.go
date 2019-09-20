package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/forget"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/gen"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/kill"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/monitor"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/node"
	"github.com/reddec/tinc-boot/cmd/tinc-boot/watch"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
)

type Config struct {
	Gen     gen.Cmd     `command:"gen" description:"Generate new tinc node over bootnode"`
	Node    node.Cmd    `command:"bootnode" description:"Serve as a boot node"`
	Monitor monitor.Cmd `command:"monitor" description:"Run as a daemon for watching new subnet and provide own host key (tinc-up)"`
	Watch   watch.Cmd   `command:"watch" description:"Add new subnet to watch daemon to get it host file (subnet-up)"`
	Forget  forget.Cmd  `command:"forget" description:"Forget subnet and stop watching it (subnet-down)"`
	Kill    kill.Cmd    `command:"kill" description:"Kill monitor daemon (tinc-down)"`
}

func main() {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	parser.LongDescription = fmt.Sprintf("Tinc nodes creator\n\nVersion: %s\nCommit: %s\n\nAuthor: Baryshnikov Aleksandr (reddec) <owner@reddec.net>",
		version, commit)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}
