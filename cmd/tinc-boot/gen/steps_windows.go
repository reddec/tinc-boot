package gen

import (
	"fmt"
	"os/exec"
)

func (cmd *Cmd) sayGoodBye() error {
	fmt.Println("DONE!")
	fmt.Println("installing service...")
	return exec.Command(cmd.TincBin, "-n", cmd.Network).Run()
}
