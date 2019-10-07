package monitor

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Iface    string        `long:"iface" env:"INTERFACE" description:"Interface to bind" required:"yes"`
	Dir      string        `long:"dir" env:"DIR" description:"Configuration directory" default:"."`
	Name     string        `long:"name" env:"NAME" description:"Self node name" required:"yes"`
	Port     int           `long:"port" env:"PORT" description:"Port to bind (should same for all hosts)" default:"1655"`
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"Attempt timeout" default:"30s"`
	Interval time.Duration `long:"interval" env:"INTERVAL" description:"Retry interval" default:"10s"`
	Reindex  time.Duration `long:"reindex" env:"REINDEX" description:"Reindex interval" default:"1m"`
}

func (cfg *Config) Hosts() string    { return filepath.Join(cfg.Dir, "hosts") }
func (cfg *Config) HostFile() string { return filepath.Join(cfg.Hosts(), cfg.Name) }
func (cfg *Config) TincConf() string { return filepath.Join(cfg.Dir, "tinc.conf") }
func (cfg *Config) Network() string  { return filepath.Base(cfg.Dir) }

func (cfg *Config) Binding() (string, error) {
	ief, err := net.InterfaceByName(cfg.Iface)
	if err != nil {
		return "", err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}
	return addrs[0].(*net.IPNet).IP.String() + ":" + strconv.Itoa(cfg.Port), nil
}

func (cfg Config) CreateAndRun(ctx context.Context) (*service, error) {
	bind, err := cfg.Binding()
	if err != nil {
		return nil, err
	}
	gctx, cancel := context.WithCancel(ctx)
	srv := &service{
		cfg:           cfg,
		globalContext: gctx,
		reindexEvent:  make(chan struct{}, 1),
		cancel:        cancel,
	}
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	api := srv.createAPI()

	srv.pool.Add(1)
	go func() {
		listener.Close()
		srv.pool.Done()
	}()

	srv.pool.Add(1)
	go func() {
		err := http.Serve(listener, api)
		if err != nil {
			srv.httpErr = err
			log.Println("[ERROR]", "serve failed:", err)
		}
		srv.pool.Done()
	}()

	srv.pool.Add(1)
	go func() {
		srv.reindexLoop()
		srv.pool.Done()
	}()

	return srv, nil
}

func (ms *service) WaitForFinish() error {
	ms.pool.Wait()
	return allErr(ms.httpErr)
}

type service struct {
	cfg           Config
	nodes         NodeArray
	globalContext context.Context
	pool          sync.WaitGroup
	reindexEvent  chan struct{}
	cancel        func()
	httpErr       error
}

func (ms *service) Stop() {
	ms.cancel()
	ms.pool.Wait()
}

func (ms *service) reindexLoop() {
	reindexTimer := time.NewTicker(ms.cfg.Reindex)
	defer reindexTimer.Stop()
	for {
		select {
		case <-ms.globalContext.Done():
			return
		case <-reindexTimer.C:
			if err := ms.indexConnectTo(); err != nil {
				log.Println("scheduled reindex failed:", err)
			}
		case <-ms.reindexEvent:
			if err := ms.indexConnectTo(); err != nil {
				log.Println("forced reindex failed:", err)
			}
		}
	}
}

func (ms *service) tryFetchHost(URL, node string, gctx context.Context) error {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(gctx, ms.cfg.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Println(res.Status)
		return errors.New("non-200 code")
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(ms.cfg.Hosts(), node), data, 0755)
}

func (ms *service) indexConnectTo() error {
	list, err := ioutil.ReadDir(ms.cfg.Hosts())
	if err != nil {
		return err
	}
	var publicNodes []string
	for _, entry := range list {
		if entry.IsDir() || entry.Name() == ms.cfg.Name {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(ms.cfg.Hosts(), entry.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "Address") {
			publicNodes = append(publicNodes, entry.Name())
		}
	}
	configContent, err := ioutil.ReadFile(ms.cfg.TincConf())
	if err != nil {
		return err
	}

	text := string(configContent)

	for _, publicNode := range publicNodes {
		matched, err := regexp.MatchString(`(?m)^ConnectTo[ ]*=[ ]*`+publicNode+"$", text)
		if err != nil {
			return err
		}
		if matched {
			continue
		}
		log.Println("new public node:", publicNode)
		text = "ConnectTo = " + publicNode + "\n" + text
	}

	err = ioutil.WriteFile(ms.cfg.TincConf(), []byte(text), 0755)
	if err != nil {
		return err
	}
	return nil
}

func (ms *service) requestNode(node *Node) {
	URL := "http://" + strings.Split(node.Subnet, "/")[0] + ":" + strconv.Itoa(ms.cfg.Port)
	for {
		log.Println("trying", URL)
		if err := ms.tryFetchHost(URL, node.Name, node.ctx); err != nil {
			log.Println(URL, ":", err)
		} else {
			log.Println(URL, "done")
			node.Fetched = true
			ms.askForIndex()
			return
		}
		select {
		case <-time.After(ms.cfg.Interval):
		case <-node.Done():
			return
		}
	}

}

func (ms *service) askForIndex() {
	select {
	case ms.reindexEvent <- struct{}{}:
	default:

	}
}

func allErr(errs ...error) error {
	var ans = make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			ans = append(ans, err.Error())
		}
	}
	if len(ans) != 0 {
		return errors.New(strings.Join(ans, "; "))
	}
	return nil
}
