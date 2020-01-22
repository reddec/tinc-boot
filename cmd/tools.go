package cmd

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"strconv"
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

func BindingByName(iface string, port int) (string, error) {
	ief, err := net.InterfaceByName(iface)
	if err != nil {
		return "", err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if v, ok := addr.(*net.IPNet); ok && v.IP.IsGlobalUnicast() {
			return v.IP.String() + ":" + strconv.Itoa(port), nil
		}
	}
	return "127.0.0.1:0", errors.New("unable to detect binding address")
}
