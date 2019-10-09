package gen

import (
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/phayes/permbits"
	"github.com/reddec/tinc-boot/scripts"
	"golang.org/x/crypto/chacha20poly1305"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Cmd struct {
	Network    string        `long:"network" env:"NETWORK" description:"Network name" default:"dnet"`
	Name       string        `long:"name" env:"NAME" description:"Self node name (trimmed hostname will be used if empty)"`
	Config     string        `long:"dir" env:"DIR" description:"Configuration directory" default:"/etc/tinc"`
	Token      string        `short:"t" long:"token" env:"TOKEN" description:"Authorization token (used as a encryption key)"`
	Prefix     string        `long:"prefix" env:"PREFIX" description:"Address prefix (left segments will be randomly auto generated)" default:"172.173"`
	Mask       int           `long:"mask" env:"MASK" description:"Network mask" default:"16"`
	Timeout    time.Duration `long:"timeout" env:"TIMEOUT" description:"Boot node request timeout" default:"15s"`
	Bin        string        `long:"bin" env:"BIN" description:"tinc-boot location" default:"/usr/local/bin/tinc-boot"`
	NoBinCopy  bool          `long:"no-bin-copy" env:"NO_BIN_COPY" description:"Disable copy tinc-boot binary"`
	NoGenKey   bool          `long:"no-gen-key" env:"NO_GEN_KEY" description:"Disable key generation"`
	Port       int           `long:"port" env:"PORT" description:"Node port (first available will be got if not set)"`
	Public     []string      `short:"a" alias:"addr" long:"public" env:"PUBLIC" description:"Public addresses that could be used for incoming connections"`
	Standalone bool          `long:"standalone" env:"STANDALONE" description:"Do not use bootnodes (usefull for very-very first initialization)"`
	Args       struct {
		URLs []string `description:"boot node urls"`
	} `positional-args:"yes"`
}

func (cmd *Cmd) Dir() string      { return filepath.Join(cmd.Config, cmd.Network) }
func (cmd *Cmd) Hosts() string    { return filepath.Join(cmd.Dir(), "hosts") }
func (cmd *Cmd) HostFile() string { return filepath.Join(cmd.Hosts(), cmd.Name) }
func (cmd *Cmd) TincConf() string { return filepath.Join(cmd.Dir(), "tinc.conf") }

func (cmd *Cmd) generateIP() (net.IP, error) {
	parts := strings.Split(cmd.Prefix, ".")
	if len(parts) > 4 {
		return nil, errors.New("invalid prefix")
	}
	var ip [net.IPv4len]byte
	for i, part := range parts {
		segment, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid prefix: %v", err)
		}
		ip[i] = byte(segment)
	}
	rand.Seed(time.Now().UnixNano())
	for i := len(parts); i < 4; i++ {
		var segment int
		if i < 3 {
			segment = rand.Intn(254)
		} else {
			segment = 1 + rand.Intn(253)
		}
		ip[i] = byte(segment)
	}
	return net.IP(ip[:]), nil
}

func (cmd *Cmd) Execute(args []string) error {
	if !cmd.Standalone && (cmd.Token == "" || len(cmd.Args.URLs) == 0) {
		log.Fatal("--token and at least one bootnode URL is required")
		return nil
	}
	if cmd.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		cmd.Name = hostname
	}

	absBin, err := filepath.Abs(cmd.Bin)
	if err != nil {
		return err
	}
	cmd.Bin = absBin

	disabledSymbols := regexp.MustCompile(`[^a-z0-9]+`)

	cmd.Network = disabledSymbols.ReplaceAllString(strings.ToLower(cmd.Network), "")
	cmd.Name = disabledSymbols.ReplaceAllString(strings.ToLower(cmd.Name), "")
	ip, err := cmd.generateIP()
	if err != nil {
		return err
	}
	// get random available port or check provided
	sock, err := net.Listen("tcp", ":"+strconv.Itoa(cmd.Port))
	if err != nil {
		return err
	}
	tSock := sock.(*net.TCPListener)
	addr := tSock.Addr()
	sock.Close()
	tAddr := addr.(*net.TCPAddr)
	cmd.Port = tAddr.Port

	fmt.Println("Network:", cmd.Network)
	fmt.Println("Name:", cmd.Name)
	fmt.Println("Subnet:", ip.String()+"/32")
	fmt.Println("Address:", ip.String()+"/"+strconv.Itoa(cmd.Mask))
	fmt.Println("Port:", cmd.Port)
	fmt.Println("Location:", cmd.Dir())

	if err := cmd.copyBinary(); err != nil {
		return err
	}

	if err := os.MkdirAll(cmd.Dir(), 0755); err != nil {
		return err
	}

	err = cmd.script("tinc-up")(&scripts.TincUpParam{
		Addr: ip.String(),
		Mask: cmd.Mask,
		Bin:  cmd.Bin,
	})
	if err != nil {
		return err
	}
	err = cmd.script("tinc-down")(&scripts.TincDownParam{
		Addr: ip.String(),
		Mask: cmd.Mask,
		Bin:  cmd.Bin,
	})
	if err != nil {
		return err
	}
	err = cmd.script("subnet-up")(&scripts.SubnetUpParam{
		Bin: cmd.Bin,
	})
	if err != nil {
		return err
	}
	err = cmd.script("subnet-down")(&scripts.SubnetDownParam{
		Bin: cmd.Bin,
	})
	if err != nil {
		return err
	}
	err = cmd.file("tinc.conf")(&scripts.TincConfParam{
		Name: cmd.Name,
		Net:  cmd.Network,
		Port: cmd.Port,
	})
	if err != nil {
		return err
	}
	err = cmd.file(filepath.Join("hosts", cmd.Name))(&scripts.HostParam{
		Port:    cmd.Port,
		Address: ip.String(),
		Public:  cmd.Public,
	})
	if err != nil {
		return err
	}
	err = cmd.runKeyGen()
	if err != nil {
		return err
	}

	err = cmd.boot()
	if err != nil {
		return err
	}
	fmt.Println("DONE!")
	fmt.Println("invoke command by root:")
	fmt.Println("")
	fmt.Println("     systemctl start tinc@" + cmd.Network)
	fmt.Println("     systemctl enable tinc@" + cmd.Network)
	fmt.Println("")
	return nil
}

func (cmd *Cmd) boot() error {
	if cmd.Standalone {
		return nil
	}
	tokenData := sha256.Sum256([]byte(cmd.Token)) // normalize to 32 bytes
	crypter, err := chacha20poly1305.NewX(tokenData[:])
	if err != nil {
		return err
	}
	decrypted, err := ioutil.ReadFile(cmd.HostFile())
	if err != nil {
		return err
	}

	var nounce [chacha20poly1305.NonceSizeX]byte
	for i := 0; i < chacha20poly1305.NonceSizeX; i++ {
		nounce[i] = byte(rand.Intn(255))
	}

	encrypted := crypter.Seal(nil, nounce[:], decrypted, []byte(cmd.Name))

	for _, URL := range cmd.Args.URLs {
		log.Println("trying", URL)
		err := cmd.requestBootnode(URL, nounce[:], encrypted, crypter)
		if err != nil {
			log.Println(URL, err)
		} else {
			return nil
		}
	}
	return errors.New("all boot nodes failed")
}

func (cmd *Cmd) runKeyGen() error {
	if cmd.NoGenKey {
		return nil
	}
	keyCmd := exec.Command("tincd", "-c", cmd.Dir(), "-K", "4096")
	keyCmd.Stdin = bytes.NewBufferString("\n\n")
	keyCmd.Stdout = os.Stdout
	keyCmd.Stderr = os.Stderr
	return keyCmd.Run()
}

func (cmd *Cmd) copyBinary() error {
	if cmd.NoBinCopy {
		return nil
	}
	err := os.MkdirAll(filepath.Dir(cmd.Bin), 0755)
	if err != nil {
		return err
	}
	cmdPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		return err
	}

	if cmdPath == cmd.Bin {
		return nil
	}

	src, err := os.Open(cmdPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(cmd.Bin)
	if err != nil {
		return err
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		return err
	}
	dest.Close()
	permissions, err := permbits.Stat(cmd.Bin)
	if err != nil {
		return err
	}
	permissions.SetGroupExecute(true)
	permissions.SetUserExecute(true)
	permissions.SetOtherExecute(true)
	return permbits.Chmod(cmd.Bin, permissions)
}

type Render interface {
	Render() ([]byte, error)
}

func (cmd *Cmd) script(name string) func(render Render) error {
	filename := filepath.Join(cmd.Dir(), name)
	return func(render Render) error {
		err := os.MkdirAll(filepath.Dir(filename), 0755)
		if err != nil {
			return err
		}
		data, err := render.Render()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filename, data, 0755)
		if err != nil {
			return err
		}
		permissions, err := permbits.Stat(filename)
		if err != nil {
			return err
		}
		permissions.SetGroupExecute(true)
		permissions.SetUserExecute(true)
		permissions.SetOtherExecute(true)
		return permbits.Chmod(filename, permissions)
	}
}

func (cmd *Cmd) file(name string) func(render Render) error {
	filename := filepath.Join(cmd.Dir(), name)
	return func(render Render) error {
		err := os.MkdirAll(filepath.Dir(filename), 0755)
		if err != nil {
			return err
		}
		data, err := render.Render()
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filename, data, 0755)
	}
}

func (cmd *Cmd) requestBootnode(URL string, nounce []byte, encryptedPayload []byte, crypter cipher.AEAD) error {
	if !strings.Contains(URL, "://") {
		URL = "http://" + URL
	}
	nounceHex := hex.EncodeToString(nounce)
	URL = URL + "/" + nounceHex + "/" + cmd.Name
	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(encryptedPayload))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New(string(data))
	}

	nodeName := res.Header.Get("X-Node")

	decrypted, err := crypter.Open(nil, nounce, data, []byte(nodeName))
	if err != nil {
		return err
	}
	log.Println("bootnode is", nodeName)

	err = ioutil.WriteFile(filepath.Join(cmd.Hosts(), nodeName), decrypted, 0755)
	if err != nil {
		return err
	}

	// prepend bootnode

	conf, err := ioutil.ReadFile(cmd.TincConf())
	if err != nil {
		return err
	}
	conf = []byte("ConnectTo = " + nodeName + "\n" + string(conf))
	return ioutil.WriteFile(cmd.TincConf(), conf, 0755)
}
