package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/reddec/tinc-boot/tincd/daemon"
)

func NewClient(ssd *SSD, config *daemon.Config, interval time.Duration) *Client {
	return &Client{
		ssd:      ssd,
		config:   config,
		interval: interval,
	}
}

type Client struct {
	ssd        *SSD
	config     *daemon.Config
	requesters map[string]*requester
	lock       sync.Mutex
	interval   time.Duration
}

func (cl *Client) Watch(ctx context.Context, address string) bool {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	if cl.requesters == nil {
		cl.requesters = make(map[string]*requester)
	}
	_, hasOld := cl.requesters[address]
	if hasOld {
		return false
	}

	child, cancel := context.WithCancel(ctx)

	rq := &requester{
		address: address,
		cancel:  cancel,
		done:    make(chan struct{}),
		ssd:     cl.ssd,
		config:  cl.config,
	}
	cl.requesters[address] = rq
	go rq.runLoop(child, cl.interval)
	return true
}

func (cl *Client) Forget(address string) {
	cl.lock.Lock()
	req, ok := cl.requesters[address]
	if !ok {
		cl.lock.Unlock()
		return
	}
	req.cancel()
	delete(cl.requesters, address)
	cl.lock.Unlock()
	<-req.done
}

func (cl *Client) Close() {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	for _, req := range cl.requesters {
		req.cancel()
	}
	for _, req := range cl.requesters {
		<-req.done
	}
	cl.requesters = nil
}

type requester struct {
	address string
	cancel  func()
	done    chan struct{}
	ssd     *SSD
	config  *daemon.Config
}

func (rq *requester) runLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer close(rq.done)
	defer ticker.Stop()
	for {
		err := rq.gatherInfo(ctx)
		if err != nil {
			log.Println("failed save meta data:", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (rq *requester) gatherInfo(ctx context.Context) error {
	entities, err := rq.fetchHeaders(ctx)
	if err != nil {
		return fmt.Errorf("fetch headers: %w", err)
	}
	var changed = false
	for _, entity := range entities {
		if !rq.ssd.CanBeMerged(entity) {
			continue
		}

		content, info, err := rq.fetchContent(ctx, entity)
		if err != nil {
			log.Println("failed get content for", entity.Name, ":", err)
			continue
		}
		log.Println("discovered node", info.Name, "version", info.Version, "from", rq.address)
		changed = changed || rq.ssd.ReplaceIfNewer(*info, func() bool {
			err = rq.config.AddHost(info.Name, content)
			if err != nil {
				log.Println("failed save file", info.Name, ":", err)
				return false
			}
			return true
		})
	}
	if !changed {
		return nil
	}
	err = rq.ssd.Save()
	if err != nil {
		return fmt.Errorf("save meta data: %w", err)
	}
	return nil
}

func (rq *requester) fetchContent(global context.Context, entity Entity) ([]byte, *Entity, error) {

	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(global, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+rq.address+"/host/"+entity.Name+"?after=-1", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("returned unexpected status code %d", res.StatusCode)
	}

	name := res.Header.Get("X-Name")
	if name == "" {
		return nil, nil, fmt.Errorf("empty name")
	}
	version, err := strconv.ParseInt(res.Header.Get("X-Version"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("parse version: %w", err)
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch content: %w", err)
	}

	newEntity := Entity{
		Name:    name,
		Version: version,
	}

	return content, &newEntity, nil
}

func (rq *requester) fetchHeaders(global context.Context) ([]Entity, error) {
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(global, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+rq.address+"/hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("returned unexpected status code %d", res.StatusCode)
	}

	var item []Entity
	err = json.NewDecoder(res.Body).Decode(&item)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return item, nil
}
