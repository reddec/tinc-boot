package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/reddec/tinc-boot/tincd/daemon/utils"
)

// Default tinc daemon configuration.
func Default(configDir string) *Config {
	return &Config{
		Binary:          "tincd",
		ConfigDir:       configDir,
		WorkDir:         filepath.Dir(configDir),
		RestartInterval: 5 * time.Second,
	}
}

// Configuration for daemons.
type Config struct {
	Binary          string   // tincd binary
	Args            []string // additional tincd arguments
	WorkDir         string   // daemon working directory
	ConfigDir       string
	RestartInterval time.Duration // interval between restart

	events Events // base events emitter that will be propagated to spawned daemons
}

// Events listeners which will be propagated to spawned daemons.
func (dm *Config) Events() *Events {
	return &dm.events
}

// Create new named daemon but not start. Name just for logs.
// To prevent go-routing leaks caller must call Stop() to cleanup resources.
func (dm *Config) Spawn(ctx context.Context, name string) *Daemon {
	child, cancel := context.WithCancel(ctx)
	d := &Daemon{
		name:   name,
		config: dm,
		cancel: cancel,
		done:   make(chan struct{}),
		status: StatusInit,
	}
	d.events.SubnetAdded.handlers = append(d.events.SubnetAdded.handlers, dm.events.SubnetAdded.handlers...)
	d.events.SubnetRemoved.handlers = append(d.events.SubnetRemoved.handlers, dm.events.SubnetRemoved.handlers...)
	go d.runLoop(child)
	return d
}

// Keygen runs tincd executable with -K flag to generate keys.
func (dm *Config) Keygen(ctx context.Context, bits int) error {
	return exec.CommandContext(ctx, dm.Binary, dm.args("-K", strconv.Itoa(bits))...).Run()
}

func (dm *Config) args(extra ...string) []string {
	var result = []string{"-D", "-d", "-d", "-d", "-d", "--pidfile", "pid.run", "-c", dm.ConfigDir}
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
	name   string
	config *Config
	cancel func()
	status Status
	done   chan struct{}
	events Events
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
	if err := os.MkdirAll(dm.config.WorkDir, 0655); err != nil {
		return fmt.Errorf("create working directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, dm.config.Binary, dm.config.args()...)
	cmd.Dir = dm.config.WorkDir
	utils.SetCmdAttrs(cmd)

	dm.setStatus(StatusRunning)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run service: %w", err)
	}
	return nil
}
