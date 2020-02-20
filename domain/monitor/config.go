package monitor

import (
	cmd2 "github.com/reddec/tinc-boot/cmd"
	"os"
	"path/filepath"
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
	var dir = cfg.Dir
	if cfg.Dir == "." {
		if d, err := os.Getwd(); err == nil {
			dir = d
		}
	}
	r, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	return r
}
func (cfg *Config) Hosts() string            { return filepath.Join(cfg.Root(), "hosts") }
func (cfg *Config) HostFile() string         { return filepath.Join(cfg.Hosts(), cfg.Name) }
func (cfg *Config) TincConf() string         { return filepath.Join(cfg.Root(), "tinc.conf") }
func (cfg *Config) Network() string          { return filepath.Base(cfg.Root()) }
func (cfg *Config) Binding() (string, error) { return cmd2.BindingByName(cfg.Iface, cfg.Port) }
