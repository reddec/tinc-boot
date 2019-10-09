package generator

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/reddec/tinc-boot/scripts"
	"io/ioutil"
	"math/rand"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defMask     = 16
	defNetwork  = "dnet"
	defPlatform = "amd64"
	defPrefix   = "172.173"
	defKeyBits  = 4096
)

const (
	minimalPort   = 1024
	filterPattern = `[^a-z0-9]+`
)

type Config struct {
	Platform string   `form:"platform"`
	Network  string   `form:"network"`
	Name     string   `form:"name"`
	Config   string   `form:"config"`
	Prefix   string   `form:"prefix"`
	Mask     int      `form:"mask"`
	Port     int      `form:"port"`
	KeyBits  int      `form:"keybits"`
	Public   []string `form:"public"`
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
		cfg.Mask = defMask
	}
	if cfg.Network == "" {
		cfg.Network = defNetwork
	}
	if cfg.Platform == "" {
		cfg.Platform = defPlatform
	}
	if cfg.Prefix == "" {
		cfg.Prefix = defPrefix
	}
	if cfg.KeyBits == 0 {
		cfg.KeyBits = defKeyBits
	}
	disabledSymbols := regexp.MustCompile(filterPattern)

	cfg.Network = disabledSymbols.ReplaceAllString(strings.ToLower(cfg.Network), "")
	cfg.Name = disabledSymbols.ReplaceAllString(strings.ToLower(cfg.Name), "")
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
