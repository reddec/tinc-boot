package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/reddec/tinc-boot/tincd/config"
	"github.com/reddec/tinc-boot/tincd/daemon/utils"
)

// Default tinc daemon configuration.
func Default(configDir string) *Config {
	return &Config{
		Binary:          "tincd",
		ConfigDir:       configDir,
		PidFile:         filepath.Join(configDir, "pid.run"),
		RestartInterval: 5 * time.Second,
	}
}

// Configuration for daemons.
type Config struct {
	Binary          string   // tincd binary
	Args            []string // additional tincd arguments
	PidFile         string
	ConfigDir       string
	RestartInterval time.Duration // interval between restart

	events Events // base events emitter that will be propagated to spawned daemons
}

// Events listeners which will be propagated to spawned daemons.
func (dm *Config) Events() *Events {
	return &dm.events
}

// Location of hosts definitions files.
func (dm *Config) HostsDir() string {
	return filepath.Join(dm.ConfigDir, "hosts")
}

// Configured daemon or not.
func (dm *Config) Configured() bool {
	main, node, err := config.ReadNodeConfig(dm.ConfigDir)
	if err != nil {
		return false
	}
	if main.Interface == "" {
		return false
	}
	ip := strings.TrimSpace(strings.Split(node.Subnet, "/")[0])
	if ip == "" {
		return false
	}
	return true
}

// Create new named daemon but not start. Name just for logs.
// To prevent go-routing leaks caller must call Stop() to cleanup resources.
func (dm *Config) Spawn(ctx context.Context) (*Daemon, error) {
	main, node, err := config.ReadNodeConfig(dm.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("read daemon config: %w", err)
	}
	if main.Interface == "" {
		return nil, fmt.Errorf("device name not defined in main config")
	}
	ip := strings.TrimSpace(strings.Split(node.Subnet, "/")[0])
	if ip == "" {
		return nil, fmt.Errorf("subnet not defined in node config")
	}

	child, cancel := context.WithCancel(ctx)
	d := &Daemon{
		name:       main.Name,
		main:       main,
		self:       node,
		config:     dm,
		cancel:     cancel,
		done:       make(chan struct{}),
		status:     StatusInit,
		ip:         ip,
		deviceName: main.Interface,
	}
	d.events.SubnetAdded.handlers = append(d.events.SubnetAdded.handlers, dm.events.SubnetAdded.handlers...)
	d.events.SubnetRemoved.handlers = append(d.events.SubnetRemoved.handlers, dm.events.SubnetRemoved.handlers...)
	d.events.Ready.handlers = append(d.events.Ready.handlers, dm.events.Ready.handlers...)
	d.events.Stopped.handlers = append(d.events.Stopped.handlers, dm.events.Stopped.handlers...)
	d.events.Configured.handlers = append(d.events.Configured.handlers, dm.events.Configured.handlers...)
	go d.runLoop(child)
	return d, nil
}

// Keygen runs tincd executable with -K flag to generate keys.
func (dm *Config) Keygen(ctx context.Context, bits int) error {
	return exec.CommandContext(ctx, dm.Binary, dm.args("-K", strconv.Itoa(bits))...).Run()
}

func (dm *Config) args(extra ...string) []string {
	var result = []string{"-D", "-d", "-d", "-d", "-d", "--pidfile", dm.PidFile, "-c", dm.ConfigDir}
	result = append(result, extra...)
	result = append(result, dm.Args...)
	return result
}

type Status string

const (
	StatusInit       = "initializing"
	StatusPending    = "pending"
	StatusRunning    = "running"
	StatusRestarting = "restarting"
	StatusStopped    = "stopped"
)

// Daemon definition. Once spawned it will restart on every failure till Stop() will be called.
// It's impossible to restart same daemon again. To recreate daemon with exactly same parameters use:
// daemon.Config().Spawn(ctx, daemon.Name()).
type Daemon struct {
	name       string
	config     *Config
	self       *config.Node
	main       *config.Main
	ip         string
	deviceName string
	cancel     func()
	status     Status
	done       chan struct{}
	events     Events
}

// Events from daemon.
func (dm *Daemon) Events() *Events {
	return &dm.events
}

// Stop and wait for finish.
func (dm *Daemon) Stop() {
	dm.cancel()
	<-dm.done
}

// Done signal.
func (dm *Daemon) Done() <-chan struct{} {
	return dm.done
}

// Config used for daemon creation. Read-only.
func (dm *Daemon) Config() *Config {
	return dm.config
}

// Self node config.
func (dm *Daemon) Self() config.Node {
	return *dm.self
}

// Main tinc config.
func (dm *Daemon) Main() config.Main {
	return *dm.main
}

// Name of daemon same as provided during creation.
func (dm *Daemon) Name() string {
	return dm.name
}

func (dm *Daemon) setStatus(status Status) {
	log.Println("daemon", dm.name, "status:", status)
	dm.status = status
}

func (dm *Daemon) runLoop(ctx context.Context) {
	defer close(dm.done)
	defer dm.setStatus(StatusStopped)
	for {
		dm.setStatus(StatusPending)
		err := dm.run(ctx)
		if err != nil {
			log.Println("daemon", dm.name, err)
		}
		dm.setStatus(StatusRestarting)
		select {
		case <-ctx.Done():
			return
		case <-time.After(dm.config.RestartInterval):
		}
	}
}

func (dm *Daemon) run(ctx context.Context) error {
	reader, writer := io.Pipe()
	cmd := exec.CommandContext(ctx, dm.config.Binary, dm.config.args()...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	utils.SetCmdAttrs(cmd)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		dm.scanner(reader)
	}()

	defer wg.Wait()
	defer writer.Close()

	defer dm.events.Stopped.emit(Configuration{
		IP:        dm.ip,
		Interface: dm.deviceName,
		Self:      *dm.self,
		Main:      *dm.main,
	})

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run service: %w", err)
	}

	return nil
}

func (dm *Daemon) scanner(stream io.Reader) {
	reader := bufio.NewScanner(stream)
	for reader.Scan() {
		line := reader.Text()
		if event := IsSubnetAdded(line); event != nil {
			dm.events.SubnetAdded.emit(*event)
		} else if event := IsSubnetRemoved(line); event != nil {
			dm.events.SubnetRemoved.emit(*event)
		} else if event := IsReady(line); event != nil {
			dm.events.Ready.emit()
			if err := dm.setupNetwork(); err != nil {
				log.Println("daemon", dm.name, "setup network:", err)
			} else {
				dm.events.Configured.emit(Configuration{
					IP:        dm.ip,
					Interface: dm.deviceName,
					Self:      *dm.self,
					Main:      *dm.main,
				})
			}
			dm.setStatus(StatusRunning)
		}
	}
	if reader.Err() != nil {
		log.Println("daemon", dm.name, "read daemon output:", reader.Err())
	}
}

func (dm *Daemon) setupNetwork() error {
	if err := setAddress(dm.deviceName, dm.ip); err != nil {
		return fmt.Errorf("set address: %w", err)
	}
	if err := enableInterface(dm.deviceName); err != nil {
		return fmt.Errorf("bring interface up: %w", err)
	}
	return nil
}

// event:"Configured"
// event:"Stopped"
type Configuration struct {
	IP        string
	Interface string
	Self      config.Node
	Main      config.Main
}
