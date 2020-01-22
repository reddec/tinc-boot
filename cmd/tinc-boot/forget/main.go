package forget

import (
	"bytes"
	"encoding/json"
	"errors"
	cmd2 "github.com/reddec/tinc-boot/cmd"
	"github.com/reddec/tinc-boot/types"
	"net/http"
)

type Cmd struct {
	Iface  string `long:"iface" env:"INTERFACE" description:"RPC interface" required:"yes"`
	Port   int    `long:"port" env:"PORT" description:"RPC port" default:"1655"`
	Subnet string `long:"subnet" env:"SUBNET" description:"Subnet address to forget" required:"yes"`
	Node   string `long:"node" env:"NODE" description:"Subnet owner name" required:"yes"`
}

func (cmd *Cmd) Execute(args []string) error {
	rpcAddr, err := cmd.binding()
	if err != nil {
		return err
	}
	var subnet = types.Subnet{
		Subnet: cmd.Subnet,
		Node:   cmd.Node,
	}
	data, err := json.Marshal(subnet)
	if err != nil {
		return err
	}
	res, err := http.Post("http://"+rpcAddr+"/rpc/forget", "application/json", bytes.NewBuffer(data))
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
