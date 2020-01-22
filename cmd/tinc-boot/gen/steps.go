// +build !windows

package gen

import "fmt"

func (cmd *Cmd) sayGoodBye() error {
	fmt.Println("DONE!")
	fmt.Println("invoke command by root:")
	fmt.Println("")
	fmt.Println("     systemctl start tinc@" + cmd.Network)
	fmt.Println("     systemctl enable tinc@" + cmd.Network)
	fmt.Println("")
	return nil
}
