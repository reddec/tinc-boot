package utils

import (
	"os/exec"
	"syscall"
)

func SetCmdAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
