package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Main struct {
	Name           string
	Port           uint16
	LocalDiscovery bool
	Interface      string
	ConnectTo      []string
}

type Node struct {
	Subnet    string
	Address   []string
	Port      uint16
	PublicKey string `tinc:"RSA PUBLIC KEY"`
}

func SaveFile(file string, content interface{}) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	err = MarshalStream(f, content)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("close config file: %w", err)
	}
	return nil
}

func ReadFile(file string, dest interface{}) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()
	return UnmarshalStream(f, dest)
}

func ReadNodeConfig(configDir string) (*Main, *Node, error) {
	var main Main
	if err := ReadFile(filepath.Join(configDir, "tinc.conf"), &main); err != nil {
		return nil, nil, fmt.Errorf("read tinc.conf: %w", err)
	}

	var node Node
	if err := ReadFile(filepath.Join(configDir, "hosts", main.Name), &node); err != nil {
		return &main, nil, fmt.Errorf("read node file: %w", err)
	}
	return &main, &node, nil
}
