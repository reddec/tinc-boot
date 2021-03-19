package boot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/reddec/tinc-boot/tincd/daemon"
	"github.com/reddec/tinc-boot/types"
)

func NewClient(url string, config *daemon.Config, token Token) *Client {
	return &Client{
		token:  token,
		url:    url,
		config: config,
	}
}

type Client struct {
	Exchanged func(name string)
	token     Token
	url       string
	config    *daemon.Config
	name      string
}

func (cl *Client) Run(ctx context.Context, retry time.Duration) {
	for {
		err := cl.exchange(ctx)
		if err != nil {
			log.Println("failed join:", err)
		} else {
			log.Println("join complete")
			return
		}
		select {
		case <-time.After(retry):
		case <-ctx.Done():
			return
		}
	}
}

func (cl *Client) exchange(ctx context.Context) error {
	const timeout = 30 * time.Second

	name, err := cl.readName()
	if err != nil {
		return fmt.Errorf("get self node name: %w", err)
	}

	selfContent, err := ioutil.ReadFile(filepath.Join(cl.config.HostsDir(), name))
	if err != nil {
		return fmt.Errorf("read self hosts: %w", err)
	}

	env := Envelope{
		Name:   cl.name,
		Config: selfContent,
	}

	encrypted, err := env.Seal(cl.token)
	if err != nil {
		return fmt.Errorf("encrypt envelope: %w", err)
	}

	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(tctx, http.MethodPost, cl.url, bytes.NewReader(encrypted))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	encryptedArchive, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	archiveData, err := cl.token.Decrypt(encryptedArchive)
	if err != nil {
		return fmt.Errorf("decrypt data: %w", err)
	}

	var archive map[string][]byte
	err = json.Unmarshal(archiveData, &archive)
	if err != nil {
		return fmt.Errorf("decode archive: %w", err)
	}

	for name, content := range archive {
		if types.CleanString(name) != name {
			log.Println("malformed archive entry:", name)
			continue
		}
		file := filepath.Join(cl.config.HostsDir(), name)
		err = ioutil.WriteFile(file, content, 0755)
		if err != nil {
			return fmt.Errorf("import host %s: %w", name, err)
		}
		if callback := cl.Exchanged; callback != nil {
			callback(name)
		}
	}
	return nil
}

func (cl *Client) readName() (string, error) {
	if cl.name != "" {
		return cl.name, nil
	}
	main, err := cl.config.Main()
	if err != nil {
		return "", fmt.Errorf("read main config: %w", err)
	}
	cl.name = main.Name
	return cl.name, nil
}
