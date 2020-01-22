package internal

type Platform struct {
	Config  string `long:"dir" env:"DIR" description:"Configuration directory" default:"C:\\Program Files (x86)\\tinc"`
	Bin     string `long:"bin" env:"BIN" description:"tinc-boot location" default:"C:\\Program Files (x86)\\tinc\\tinc-boot.exe"`
	TincBin string `long:"tinc-bin" env:"TINC_BIN" description:"Path to tincd executable" default:"C:\\Program Files (x86)\\tinc\\tincd.exe"`
}
