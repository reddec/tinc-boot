package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/reddec/tinc-boot/tincd/boot"
	"github.com/reddec/tinc-boot/tincd/config"
	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/tincd/daemon/utils"
	"github.com/reddec/tinc-boot/tincd/discovery"
	"github.com/reddec/tinc-boot/types"
)

type Cmd struct {
	Name              string        `short:"n" long:"name" env:"NAME" description:"Node name. If not set - hostname with random suffix will be used"`
	Advertise         []string      `short:"a" long:"advertise" env:"ADVERTISE" description:"Routable IPs/domains with or without port that will be advertised for new clients. If not set - all non-loopback IPs will be used"`
	TincPort          uint16        `long:"tinc-port" env:"TINC_PORT" description:"Tinc listen port for fresh node. If not set - random will be generated in 30000-40000 range"`
	Device            string        `long:"device" env:"DEVICE" description:"Device name. If not defined - will use last 5 symbols of resolved name"`
	Port              uint16        `short:"p" long:"port" env:"PORT" description:"Greeting service binding port" default:"8655"`
	Host              string        `short:"h" long:"host" env:"HOST" description:"Greeting service binding host" default:""`
	Token             string        `short:"t" long:"token" env:"TOKEN" description:"Boot token. If not defined - random string will be generated and printed"`
	TLS               bool          `long:"tls" env:"TLS" description:"Enable TLS for greeting protocol"`
	Cert              string        `long:"cert" env:"CERT" description:"TLS certificate" default:"server.crt"`
	Key               string        `long:"key" env:"KEY" description:"TLS key" default:"server.key"`
	IP                string        `long:"ip" env:"IP" description:"VPN IP for fresh node. If not set - random will be generated once in 172.16.0.0/12"`
	Dir               string        `short:"d" long:"dir" env:"DIR" description:"tinc-boot directory. Will be created if not exists" default:"vpn"`
	Tincd             string        `long:"tincd" env:"TINCD" description:"tincd binary location" default:"tincd"`
	Join              []string      `short:"j" long:"join" env:"JOIN" description:"URLs to join to another network"`
	JoinRetry         time.Duration `long:"join-retry" env:"JOIN_RETRY" description:"Retry interval" default:"15s"`
	DiscoveryInterval time.Duration `long:"discovery-interval" env:"DISCOVERY_INTERVAL" description:"Interval between discovery" default:"5s"`
	UFW               bool          `long:"ufw" env:"UFW" description:"Open ports using ufw" `
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

func (cmd Cmd) tincPort() uint16 {
	if cmd.TincPort != 0 {
		return cmd.TincPort

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

	return net.IPv4(172, 16 + byte(rand.Intn(15)), byte(rand.Intn(255)), 1+byte(rand.Intn(254))).String()
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

	daemonConfig := daemon.Default(cmd.configDir())
	daemonConfig.PidFile = filepath.Join(cmd.workDir(), "pid.run")

	ssd := discovery.NewSSD(cmd.ssdFile())

	if err := ssd.Read(); err != nil {
		return fmt.Errorf("read discovery: %w", err)
	}

	// restore SSD config if we missed something by scanning hosts
	hosts, err := daemonConfig.HostNames()
	if err != nil {
		return fmt.Errorf("read hosts: %w", err)
	}
	for _, host := range hosts {
		ssd.ReplaceIfNewer(discovery.Entity{
			Name: host,
		}, nil)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	if cmd.UFW {
		cmd.automaticFirewall(ctx, daemonConfig)
	}

	// configure daemon if needed
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

	// add to discovery information about self node
	ssd.Replace(discovery.Entity{
		Name:    main.Name,
		Version: tick,
	})

	err = ssd.Save() // replace self discovery
	if err != nil {
		log.Println("save discovery meta config (fallback to in-memory only):", err)
	}

	// re-index config
	err = daemonConfig.IndexHosts()
	if err != nil {
		return fmt.Errorf("index hosts: %w", err)
	}

	discoveryService := discovery.New(ssd, daemonConfig, cmd.DiscoveryInterval)

	daemonConfig.Events().SubscribeAll(discoveryService)

	instance, err := daemonConfig.Spawn(ctx)
	if err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	defer instance.Stop()

	// setup boot/greeting service
	if cmd.Token == "" {
		cmd.Token = utils.RandStringRunes(64)
	}
	var proto = "http"
	if cmd.TLS {
		proto = "https"
	}
	port := fmt.Sprint(cmd.Port) // TODO: replace to listener Listen and get real port
	var lines = []string{
		"Use one of this commands to join the network",
		"",
	}
	for _, address := range cmd.advertise() {
		lines = append(lines, os.Args[0]+" run -t "+cmd.Token+" --join "+proto+"://"+address+":"+port)
	}
	fmt.Println(strings.Join(lines, "\n"))

	token := boot.Token(cmd.Token)
	// setup greeting clients
	var greetClients sync.WaitGroup
	for _, url := range cmd.Join {
		client := boot.NewClient(url, daemonConfig, token)
		client.Exchanged = func(name string) {
			if ssd.ReplaceIfNewer(discovery.Entity{
				Name: name,
			}, nil) {
				log.Println("got new node", name, "from", url)
			}
			if err := ssd.Save(); err != nil {
				log.Println("failed save discovery metadata after exchange:", err)
			}
		}
		client.Complete = func() {
			instance.Reload()
		}
		greetClients.Add(1)
		go func(client *boot.Client) {
			defer greetClients.Done()
			client.Run(ctx, cmd.JoinRetry)
		}(client)
	}

	// setup own greeting service
	greetHandler := boot.NewServer(daemonConfig, token)
	greetHandler.Joined = func(info boot.Envelope) {
		// refresh discovery
		if ssd.ReplaceIfNewer(discovery.Entity{
			Name:    info.Name,
			Version: 0,
		}, nil) {
			instance.Reload()
		}
		if err := ssd.Save(); err != nil {
			log.Println("failed save discovery metadata:", err)
		}
	}

	greetServer := &http.Server{
		Addr:    fmt.Sprint(cmd.Host, ":", cmd.Port),
		Handler: greetHandler,
	}

	go func() {
		<-ctx.Done()
		_ = greetServer.Close()
	}()

	if cmd.TLS {
		err = greetServer.ListenAndServeTLS(cmd.Cert, cmd.Key)
	} else {
		err = greetServer.ListenAndServe()
	}
	if err != nil {
		log.Println(err)
	}

	cancel()
	<-instance.Done()
	greetClients.Wait()
	return nil
}

func (cmd Cmd) createConfig(ctx context.Context, daemonConfig *daemon.Config) error {
	var main = config.Main{
		Name:           cmd.name(),
		Port:           cmd.tincPort(),
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

func (cmd Cmd) automaticFirewall(ctx context.Context, dc *daemon.Config) {
	dc.Events().Configured.Subscribe(func(configuration daemon.Configuration) {
		if err := exec.CommandContext(ctx, "ufw", "allow", fmt.Sprint(configuration.Main.Port)).Run(); err != nil {
			log.Println("failed allow incoming requests for traffic:", err)
		} else {
			log.Println("opened incoming port", fmt.Sprint(configuration.Main.Port))
		}
		if err := exec.CommandContext(ctx, "ufw", "allow", "from", "any", "to", "any", "port", fmt.Sprint(cmd.Port), "proto", "tcp").Run(); err != nil {
			log.Println("failed allow incoming requests on boot port:", err)
		} else {
			log.Println("opened boot port", fmt.Sprint(cmd.Port))
		}
		if err := exec.CommandContext(ctx, "ufw", "allow", "in", "on", configuration.Interface, "to", "any", "port", discovery.Port, "proto", "tcp").Run(); err != nil {
			log.Println("failed allow internal ports for discovery:", err)
		} else {
			log.Println("opened discovery port", discovery.Port, "on", configuration.Interface)
		}
	})
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
