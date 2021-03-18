package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/reddec/tinc-boot/tincd/config"
	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/tincd/daemon/utils"
	"github.com/reddec/tinc-boot/tincd/discovery"
	"github.com/reddec/tinc-boot/types"
)

type Cmd struct {
	Name              string        `short:"n" long:"name" env:"NAME" description:"Node name. If not set - saved or hostname with random suffix will be used"`
	Advertise         []string      `short:"a" long:"advertise" env:"ADVERTISE" description:"Routable IPs/domains with or without port that will be advertised for new clients. If not set - saved or all non-loopback IPs will be used"`
	Port              uint16        `short:"p" long:"port" env:"PORT" description:"Tinc listen port for fresh node. If not set - saved or random will be generated in 30000-40000 range"`
	Device            string        `long:"device" env:"DEVICE" description:"Device name. If not defined - will use last 5 symbols of resolved name"`
	IP                string        `long:"ip" env:"IP" description:"VPN IP for fresh node. If not set - saved or random will be generated once in 172.0.0.0/8"`
	Dir               string        `short:"d" long:"dir" env:"DIR" description:"tinc-boot directory. Will be created if not exists" default:"vpn"`
	Tincd             string        `long:"tincd" env:"TINCD" description:"tincd binary location" default:"tincd"`
	DiscoveryInterval time.Duration `long:"discovery-interval" env:"DISCOVERY_INTERVAL" description:"Interval between discovery" default:"5s"`
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

func (cmd Cmd) ssdFile() string {
	return filepath.Join(cmd.workDir(), "discovery.json")
}

func (cmd Cmd) clockFile() string {
	return filepath.Join(cmd.workDir(), "clock")
}

func (cmd Cmd) advertise() []string {
	if len(cmd.Advertise) > 0 {
		var ans = make([]string, 0, len(cmd.Advertise))
		for _, addr := range cmd.Advertise {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				log.Println("parse address", addr, ":", err)
				continue
			}
			ans = append(ans, host+" "+port)
		}
		return ans
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

func (cmd *Cmd) name() string {
	if cmd.Name != "" {
		return cmd.Name
	}
	name, _ := os.Hostname()

	cmd.Name = types.CleanString(name + utils.RandStringRunes(5))
	return cmd.Name
}

func (cmd Cmd) deviceName() string {
	if cmd.Device != "" {
		return cmd.Device
	}
	name := cmd.name()
	if len(name) <= 5 {
		return name
	}
	return name[len(name)-5:]
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

	tick, err := cmd.nextTick()
	if err != nil {
		return fmt.Errorf("count clock tick: %w", err)
	}

	ssd := discovery.NewSSD(cmd.ssdFile())

	if err := ssd.Read(); err != nil {
		return fmt.Errorf("read discovery: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	daemonConfig := daemon.Default(cmd.configDir())
	daemonConfig.PidFile = filepath.Join(cmd.workDir(), "pid.run")

	if !daemonConfig.Configured() {
		log.Println("configuration not exists or invalid - creating a new one")
		err := cmd.createConfig(ctx, daemonConfig)
		if err != nil {
			return fmt.Errorf("create config: %w", err)
		}
	} else {
		log.Println("using existent configuration")
	}

	main, _, err := config.ReadNodeConfig(daemonConfig.ConfigDir)
	if err != nil {
		return fmt.Errorf("read generated config: %w", err)
	}

	ssd.Replace(discovery.Entity{
		Name:    main.Name,
		Version: tick,
	})

	err = ssd.Save() // replace self discovery
	if err != nil {
		log.Println("save discovery meta config (fallback to in-memory only):", err)
	}

	discoveryService := discovery.New(ssd, daemonConfig, cmd.DiscoveryInterval)

	daemonConfig.Events().SubscribeAll(discoveryService)

	instance, err := daemonConfig.Spawn(ctx)
	if err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	defer instance.Stop()

	<-instance.Done()

	return nil
}

func (cmd Cmd) createConfig(ctx context.Context, daemonConfig *daemon.Config) error {
	var main = config.Main{
		Name:           cmd.name(),
		Port:           cmd.port(),
		LocalDiscovery: true,
		Interface:      "tun" + strings.ToUpper(cmd.deviceName()),
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

	if err := daemonConfig.Keygen(ctx, 4096); err != nil {
		return fmt.Errorf("generate keys: %w", err)
	}

	return nil
}

func (cmd Cmd) nextTick() (int64, error) {
	data, err := ioutil.ReadFile(cmd.clockFile())
	if os.IsNotExist(err) {
		data = []byte("0")
	} else if err != nil {
		return 0, fmt.Errorf("read clock: %w", err)
	}

	tick, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		// broken clock
		tick = 0
	}

	tick++
	return tick, ioutil.WriteFile(cmd.clockFile(), []byte(strconv.FormatInt(tick, 10)), 0755)
}

func getAllRoutableIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("get interface addrs: %w", err)
	}
	var ans []string
	for _, addr := range addrs {
		if ipaddr, ok := addr.(*net.IPNet); ok {
			if !ipaddr.IP.IsLoopback() && ipaddr.IP.IsGlobalUnicast() {
				ans = append(ans, ipaddr.IP.String())
			}
		}
	}
	return ans, nil
}
