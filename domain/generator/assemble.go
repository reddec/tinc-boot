package generator

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/reddec/tinc-boot/scripts"
	"github.com/reddec/tinc-boot/types"
	"io/ioutil"
	"math/rand"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefMask     = 16
	DefNetwork  = "dnet"
	DefPlatform = "amd64"
	DefPrefix   = "172.173"
	DefKeyBits  = 4096
)

const (
	minimalPort = 1024
)

type Config struct {
	Platform string   `form:"platform"` // optional, target platform name (amd64, arm64, ...). Used to create link to download binary
	Network  string   `form:"network"`  // optional, network name (also device name)
	Name     string   `form:"name"`     // required
	Prefix   string   `form:"prefix"`   // optional
	Mask     int      `form:"mask"`     // optional
	Port     int      `form:"port"`     // optional, default random in range 1024-65535
	KeyBits  int      `form:"keybits"`  // optional
	Public   []string `form:"public"`   // optional, list of public ip for the node
}

type Assembly struct {
	Script    []byte
	Config    Config
	PublicKey string
}

func (cfg *Config) Generate(currentNetDir string) (*Assembly, error) {
	if cfg.Name == "" {
		return nil, errors.New("name not defined")
	}
	if cfg.Mask == 0 {
		cfg.Mask = DefMask
	}
	if cfg.Network == "" {
		cfg.Network = DefNetwork
	}
	if cfg.Platform == "" {
		cfg.Platform = DefPlatform
	}
	if cfg.Prefix == "" {
		cfg.Prefix = DefPrefix
	}
	if cfg.KeyBits == 0 {
		cfg.KeyBits = DefKeyBits
	}
	cfg.Network = types.CleanString(cfg.Network)
	cfg.Name = types.CleanString(cfg.Name)
	if cfg.Name == "" {
		return nil, errors.New("name contains only disabled symbols")
	}
	ip, err := cfg.generateIP()
	if err != nil {
		return nil, err
	}
	if cfg.Port == 0 {
		cfg.Port = minimalPort + rand.Intn(65535-minimalPort)
	}
	publicNodes, err := findAllPublicNodes(filepath.Join(currentNetDir, "hosts"))
	if err != nil {
		return nil, err
	}
	keys, err := GenerateKeys(cfg.KeyBits)
	if err != nil {
		return nil, err
	}
	script := &scripts.AssemblyParam{
		Public:      cfg.Public,
		Name:        cfg.Name,
		Network:     cfg.Network,
		Address:     ip.String(),
		Mask:        cfg.Mask,
		Port:        cfg.Port,
		Platform:    cfg.Platform,
		ConnectTo:   publicNodes,
		HostPublic:  keys.Public,
		HostPrivate: keys.Private,
	}

	scriptData, err := script.Render()
	if err != nil {
		return nil, err
	}
	return &Assembly{
		Script:    scriptData,
		Config:    *cfg,
		PublicKey: keys.Public,
	}, nil
}

func (cfg *Config) generateIP() (net.IP, error) {
	parts := strings.Split(cfg.Prefix, ".")
	if len(parts) > 4 {
		return nil, errors.New("invalid prefix - too long")
	}
	if len(parts) == 1 && parts[0] == "" {
		parts = nil
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

func findAllPublicNodes(rootDir string) (map[string]string, error) {
	list, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	public := map[string]string{}
	for _, item := range list {
		if item.IsDir() {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(rootDir, item.Name()))
		if err != nil {
			return nil, err
		}
		if strings.Contains(string(content), "Address") {
			public[item.Name()] = string(content)
		}
	}
	return public, nil
}

type Keys struct {
	Private string
	Public  string
}

func GenerateKeys(bitSize int) (*Keys, error) {
	priv, err := rsa.GenerateKey(cryptorand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	return &Keys{
		Private: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		})),
		Public: string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
		})),
	}, nil
}
