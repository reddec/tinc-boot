package monitor

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/reddec/struct-view/support/events"
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

const retryInterval = 3 * time.Second

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
		address:       bind,
		cancel:        cancel,
		events:        cfg.events,
	}

	var listener net.Listener

	for {
		serverListener, err := net.Listen("tcp", bind)
		if err != nil {
			log.Println(err)
		} else {
			listener = serverListener
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
		}
	}
	api := srv.createAPI()

	// events streaming over HTTP/WS
	stream := events.NewWebsocketStream()
	srv.events.Sink(stream.Feed)
	api.GET("/rpc/ws", gin.WrapH(stream.Handler()))

	srv.pool.Add(1)
	go func() {
		<-gctx.Done()
		listener.Close()
		stream.Close()
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

type service struct {
	cfg           Config
	nodes         NodeArray
	globalContext context.Context
	pool          sync.WaitGroup
	initTemplates sync.Once
	events        monitorEvents
	reindexEvent  chan struct{}
	address       string
	cancel        func()
	httpErr       error
}

func (ms *service) WaitForFinish() error {
	ms.pool.Wait()
	return allErr(ms.httpErr)
}

func (ms *service) Events() *monitorEvents { return &ms.events }

func (ms *service) Stop() {
	ms.cancel()
	ms.pool.Wait()
}

func (ms *service) Config() Config { return ms.cfg }

func (ms *service) Address() string { return ms.address }

func (ms *service) reindexLoop() {
	reindexTimer := time.NewTicker(ms.cfg.Reindex)
	defer reindexTimer.Stop()
	ms.askForIndex()
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

func (ms *service) tryFetchHost(URL, node string, gctx context.Context) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(gctx, ms.cfg.Timeout)
	defer cancel()
	req = req.WithContext(ctx)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Println(res.Status)
		return nil, errors.New("non-200 code")
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, ioutil.WriteFile(filepath.Join(ms.cfg.Hosts(), node), data, 0755)
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
		if data, err := ms.tryFetchHost(URL, node.Name, node.ctx); err != nil {
			log.Println(URL, ":", err)
		} else {
			log.Println(URL, "done")
			node.Fetched = true
			node.Public = strings.Contains(string(data), "Address")
			ms.askForIndex()
			ms.events.Fetched.emit(node)
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
