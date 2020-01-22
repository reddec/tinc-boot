package kill

import (
	"errors"
	cmd2 "github.com/reddec/tinc-boot/cmd"
	"io"
	"net/http"
)

type Cmd struct {
	Iface string `long:"iface" env:"INTERFACE" description:"RPC interface" required:"yes"`
	Port  int    `long:"port" env:"PORT" description:"RPC port" default:"1655"`
}

func (cmd *Cmd) Execute(args []string) error {
	rpcAddr, err := cmd.binding()
	if err != nil {
		return err
	}
	res, err := http.Post("http://"+rpcAddr+"/rpc/kill", "application/json", nil)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {

		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		return errors.New(res.Status)
	}
	return nil
}

func (cmd *Cmd) binding() (string, error) {
	return cmd2.BindingByName(cmd.Iface, cmd.Port)
}
