// +build !windows

package internal

type Platform struct {
	Dir     string `long:"dir" env:"DIR" description:"Configuration directory (including net)" default:"/etc/tinc/dnet"`
	Service bool   `long:"service" env:"SERVICE" description:"Generate service file to /etc/systemd/system/tinc-boot-{net}.service"`
}
