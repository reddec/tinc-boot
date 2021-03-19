package daemon

import "os/exec"

func setAddress(interfaceName string, ip string) error {
	return exec.Command("ip", "addr", "add", ip+"/32", "dev", interfaceName).Run()
}

func enableInterface(interfaceName string) error {
	return exec.Command("ip", "link", "set", "dev", interfaceName, "up").Run()
}

func setRouting(interfaceName string, subnet string) error {
	return exec.Command("ip", "route", "add", subnet, "dev", interfaceName).Run()
}

func removeRouting(interfaceName string, subnet string) error {
	return exec.Command("ip", "route", "del", subnet, "dev", interfaceName).Run()
}
