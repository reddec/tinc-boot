package run

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/reddec/tinc-boot/tincd/config"
	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/tincd/daemon/utils"
	"github.com/reddec/tinc-boot/types"
)

type Cmd struct {
	Name      string   `long:"name" env:"NAME" description:"Node name. If not set - saved or hostname with random suffix will be used"`
	Advertise []string `short:"a" long:"advertise" env:"ADVERTISE" description:"Routable IPs or IPs with ports that will be advertised for new clients. If not set - saved or all non-loopback IPs will be used"`
	Port      uint16   `short:"p" long:"port" env:"PORT" description:"Tinc listen port for fresh node. If not set - saved or random will be generated in 30000-40000 range"`
	IP        string   `long:"ip" env:"IP" description:"VPN IP for fresh node. If not set - saved or random will be generated once in 172.0.0.0/8"`
	Dir       string   `short:"d" long:"dir" env:"DIR" description:"tinc-boot directory. Will be created if not exists" default:"vpn"`
	Tincd     string   `long:"tincd" env:"TINCD" description:"tincd binary location" default:"tincd"`
}

func (cmd Cmd) configDir() string {
	return filepath.Join(cmd.Dir, "config")
}

func (cmd Cmd) tincFile() string {
	return filepath.Join(cmd.configDir(), "tinc.conf")
}

func (cmd Cmd) hostsDir() string {
	return filepath.Join(cmd.configDir(), "hosts")
}

func (cmd Cmd) workDir() string {
	return filepath.Join(cmd.Dir, "run")
}

func (cmd Cmd) advertise() []string {
	if len(cmd.Advertise) > 0 {
		return cmd.Advertise
	}
	ips, err := getAllRoutableIPs()
	if err != nil {
		log.Println("get routable ips:", err)
	}
	return ips
}

func (cmd Cmd) port() uint16 {
	if cmd.Port != 0 {
		return cmd.Port

	}
	return uint16(30000 + rand.Intn(10000))
}

func (cmd Cmd) name() string {
	if cmd.Name != "" {
		return cmd.Name
	}
	name, _ := os.Hostname()
	return types.CleanString(name + utils.RandStringRunes(5))
}

func (cmd Cmd) ip() string {
	if cmd.IP != "" {
		return cmd.IP
	}

	return net.IPv4(172, byte(rand.Intn(255)), byte(rand.Intn(255)), 1+byte(rand.Intn(254))).String()
}

func (cmd *Cmd) Execute([]string) error {
	rand.Seed(time.Now().UnixNano())
	if err := os.MkdirAll(cmd.configDir(), 0755); err != nil {
		return fmt.Errorf("create configuration dir: %w", err)
	}
	if err := os.MkdirAll(cmd.hostsDir(), 0755); err != nil {
		return fmt.Errorf("create nodes dir: %w", err)
	}
	if err := os.MkdirAll(cmd.workDir(), 0755); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	daemonConfig := daemon.Default(cmd.configDir())
	daemonConfig.PidFile = filepath.Join(cmd.workDir(), "pid.run")

	var main = config.Main{
		Name:           cmd.name(),
		Port:           cmd.port(),
		LocalDiscovery: true,
		Interface:      "tun" + strings.ToUpper(utils.RandStringRunes(5)),
	}
	if err := config.SaveFile(cmd.tincFile(), main); err != nil {
		return fmt.Errorf("create tinc.conf file: %w", err)
	}

	nodeFile := filepath.Join(cmd.hostsDir(), main.Name)

	var node = config.Node{
		Subnet:  cmd.ip() + "/32",
		Address: cmd.advertise(),
		Port:    main.Port,
	}
	if err := config.SaveFile(nodeFile, node); err != nil {
		return fmt.Errorf("create node file: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	if err := daemonConfig.Keygen(ctx, 4096); err != nil {
		return fmt.Errorf("generate keys: %w", err)
	}

	instance, err := daemonConfig.Spawn(ctx)
	if err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	defer instance.Stop()

	<-instance.Done()

	return nil
}

func getAllRoutableIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("get interface addrs: %w", err)
	}
	var ans []string
	for _, addr := range addrs {
		if ipaddr, ok := addr.(*net.IPAddr); ok {
			if !ipaddr.IP.IsLoopback() {
				ans = append(ans, ipaddr.IP.String())
			}
		}
	}
	return ans, nil
}
