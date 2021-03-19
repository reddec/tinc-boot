//+build !linux

package run

import "fmt"

type Cmd struct {
}

func (cmd Cmd) Execute([]string) error {
	return fmt.Errorf("not implemented on the platform. Only Linux supported")
}
