package watch

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/reddec/tinc-boot/types"
	"net"
	"net/http"
	"strconv"
)

type Cmd struct {
	Iface  string `long:"iface" env:"INTERFACE" description:"RPC interface" required:"yes"`
	Port   int    `long:"port" env:"PORT" description:"RPC port" default:"1655"`
	Subnet string `long:"subnet" env:"SUBNET" description:"Subnet address to watch" required:"yes"`
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
	res, err := http.Post("http://"+rpcAddr+"/rpc/watch", "application/json", bytes.NewBuffer(data))
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
	ief, err := net.InterfaceByName(cmd.Iface)
	if err != nil {
		return "", err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}
	return addrs[0].(*net.IPNet).IP.String() + ":" + strconv.Itoa(cmd.Port), nil
}
