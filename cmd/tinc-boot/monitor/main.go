package monitor

import (
	"github.com/reddec/tinc-boot/cmd"
	"github.com/reddec/tinc-boot/domain/monitor"
)

type Cmd struct {
	monitor.Config
}

func (app *Cmd) Execute(args []string) error {
	srv, err := app.Config.CreateAndRun(cmd.SignalContext(nil))
	if err != nil {
		return err
	}
	return srv.WaitForFinish()
}
