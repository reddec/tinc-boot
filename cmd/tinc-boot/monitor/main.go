package monitor

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/reddec/tinc-boot/types"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Cmd struct {
	Iface    string        `long:"iface" env:"INTERFACE" description:"Interface to bind" required:"yes"`
	Dir      string        `long:"dir" env:"DIR" description:"Configuration directory" default:"."`
	Name     string        `long:"name" env:"NAME" description:"Self node name" required:"yes"`
	Port     int           `long:"port" env:"PORT" description:"Port to bind (should same for all hosts)" default:"1655"`
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"Attempt timeout" default:"30s"`
	Interval time.Duration `long:"interval" env:"INTERVAL" description:"Retry interval" default:"10s"`
	Reindex  time.Duration `long:"reindex" env:"REINDEX" description:"Reindex interval" default:"1m"`
}

func (cmd *Cmd) Hosts() string    { return filepath.Join(cmd.Dir, "hosts") }
func (cmd *Cmd) HostFile() string { return filepath.Join(cmd.Hosts(), cmd.Name) }
func (cmd *Cmd) TincConf() string { return filepath.Join(cmd.Dir, "tinc.conf") }
func (cmd *Cmd) Network() string  { return filepath.Base(cmd.Dir) }

func (cmd *Cmd) Execute(args []string) error {
	var watch = make(chan types.Subnet, 1)
	var forget = make(chan string, 1)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/", func(gctx *gin.Context) {
		gctx.File(cmd.HostFile())
	})
	engine.POST("/rpc/watch", func(gctx *gin.Context) {
		var subnet types.Subnet
		if err := gctx.Bind(&subnet); err != nil {
			return
		}
		watch <- subnet
		gctx.AbortWithStatus(http.StatusNoContent)
	})
	engine.POST("/rpc/forget", func(gctx *gin.Context) {
		var subnet types.Subnet
		if err := gctx.Bind(&subnet); err != nil {
			return
		}
		forget <- subnet.Node
		gctx.AbortWithStatus(http.StatusNoContent)
	})
	engine.POST("/rpc/kill", func(gctx *gin.Context) {
		gctx.AbortWithStatus(http.StatusNoContent)
		os.Exit(0)
	})
	go cmd.watchSubnets(watch, forget)

	binding, err := cmd.binding()
	if err != nil {
		return err
	}

	log.Println("RPC on", binding)
	return engine.Run(binding)
}

func (cmd *Cmd) binding() (string, error) {
	ief, err := net.InterfaceByName(cmd.Iface)
	if err != nil {
		return "", err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}
	return addrs[0].(*net.IPNet).IP.String() + ":" + strconv.Itoa(cmd.Port), nil
}

func (cmd *Cmd) watchSubnets(watch <-chan types.Subnet, forget <-chan string) {
	all := map[string]func(){}
	var reindex = make(chan struct{}, 1)
	reindex <- struct{}{}
	reindexTimer := time.NewTicker(cmd.Reindex)
	defer reindexTimer.Stop()
LOOP:
	for {
		select {
		case subnet, ready := <-watch:
			if !ready {
				break LOOP
			}
			ctx, cancel := context.WithCancel(context.Background())
			if oldCancel, ok := all[subnet.Node]; ok {
				oldCancel()
			}
			all[subnet.Node] = cancel
			go func() {
				defer cancel()
				if cmd.watchNetwork(subnet, ctx) {
					reindex <- struct{}{}
				}
			}()
		case node, ready := <-forget:
			if !ready {
				break LOOP
			}
			if oldCancel, ok := all[node]; ok {
				oldCancel()
			}
			delete(all, node)
		case <-reindexTimer.C:
			if err := cmd.indexConnectTo(); err != nil {
				log.Println("reindex:", err)
			}
		case <-reindex:
			if err := cmd.indexConnectTo(); err != nil {
				log.Println("find public nodes:", err)
			}
		}
	}
	for _, c := range all {
		c()
	}
}

func (cmd *Cmd) watchNetwork(net types.Subnet, ctx context.Context) bool {
	URL := "http://" + strings.Split(net.Subnet, "/")[0] + ":" + strconv.Itoa(cmd.Port)
	for {
		log.Println("trying", URL)
		if err := cmd.tryFetchHost(URL, net.Node, ctx); err != nil {
			log.Println(URL, ":", err)
		} else {
			log.Println(URL, "done")
			return true
		}

		select {
		case <-time.After(cmd.Interval):
		case <-ctx.Done():
			return false
		}
	}
}

func (cmd *Cmd) tryFetchHost(URL, node string, gctx context.Context) error {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(gctx, cmd.Timeout)
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
	return ioutil.WriteFile(filepath.Join(cmd.Hosts(), node), data, 0755)
}

func (cmd *Cmd) indexConnectTo() error {
	list, err := ioutil.ReadDir(cmd.Hosts())
	if err != nil {
		return err
	}
	var publicNodes []string
	for _, entry := range list {
		if entry.IsDir() || entry.Name() == cmd.Name {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(cmd.Hosts(), entry.Name()))
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "Address") {
			publicNodes = append(publicNodes, entry.Name())
		}
	}
	configContent, err := ioutil.ReadFile(cmd.TincConf())
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

	err = ioutil.WriteFile(cmd.TincConf(), []byte(text), 0755)
	if err != nil {
		return err
	}
	return nil
}
