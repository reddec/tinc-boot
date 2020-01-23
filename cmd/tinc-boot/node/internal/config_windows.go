package internal

type Platform struct {
	Dir     string `long:"dir" env:"DIR" description:"Configuration directory (including net)" default:"C:\\Program Files (x86)\\tinc\\dnet"`
	Service bool
}
