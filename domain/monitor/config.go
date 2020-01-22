package monitor

import (
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Iface    string        `long:"iface" env:"INTERFACE" description:"Interface to bind" required:"yes"`
	Dir      string        `long:"dir" env:"DIR" description:"Configuration directory" default:"."`
	Name     string        `long:"name" env:"NAME" description:"Self node name" required:"yes"`
	Port     int           `long:"port" env:"PORT" description:"Port to bind (should same for all hosts)" default:"1655"`
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"Attempt timeout" default:"30s"`
	Interval time.Duration `long:"interval" env:"INTERVAL" description:"Retry interval" default:"10s"`
	Reindex  time.Duration `long:"reindex" env:"REINDEX" description:"Reindex interval" default:"1m"`
	events   monitorEvents
}

func (cfg *Config) Events() *monitorEvents { return &cfg.events }
func (cfg *Config) Root() string {
	r, err := filepath.Abs(cfg.Dir)
	if err != nil {
		panic(err)
	}
	return r
}
func (cfg *Config) Hosts() string    { return filepath.Join(cfg.Root(), "hosts") }
func (cfg *Config) HostFile() string { return filepath.Join(cfg.Hosts(), cfg.Name) }
func (cfg *Config) TincConf() string { return filepath.Join(cfg.Root(), "tinc.conf") }
func (cfg *Config) Network() string  { return filepath.Base(cfg.Root()) }

func (cfg *Config) Binding() (string, error) {
	ief, err := net.InterfaceByName(cfg.Iface)
	if err != nil {
		return "", err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if v, ok := addr.(*net.IPNet); ok && v.IP.IsGlobalUnicast() {
			return v.IP.String() + ":" + strconv.Itoa(cfg.Port), nil
		}
	}
	return "127.0.0.1:0", errors.New("unable to detect binding address")
}
