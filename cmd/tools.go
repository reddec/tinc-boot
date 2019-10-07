package cmd

import (
	"context"
	"os"
	"os/signal"
)

func SignalContext(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	ctx, closer := context.WithCancel(parent)
	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Kill, os.Interrupt)
		for range c {
			closer()
			break
		}
	}()
	return ctx
}
