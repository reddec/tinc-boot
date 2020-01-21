// +build !windows

package internal

type Platform struct {
	Config  string `long:"dir" env:"DIR" description:"Configuration directory" default:"/etc/tinc"`
	Bin     string `long:"bin" env:"BIN" description:"tinc-boot location" default:"/usr/local/bin/tinc-boot"`
	TincBin string `long:"tinc-bin" env:"TINC_BIN" description:"Path to tincd executable" default:"tincd"`
}
