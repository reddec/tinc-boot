package monitor

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/reddec/tinc-boot/domain/generator"
	"github.com/reddec/tinc-boot/types"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

type NodeList struct {
	Nodes []*Node `json:"nodes"`
}

func (ms *service) createAPI() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/", ms.apiServeHostFile)
	engine.GET("/ui", ms.apiUiNodes)
	engine.POST("/ui", ms.apiUiAddNode)
	engine.POST("/rpc/watch", ms.apiWatchNode)
	engine.POST("/rpc/forget", ms.apiForgetNode)
	engine.POST("/rpc/kill", ms.apiKillNode)
	engine.GET("/rpc/nodes", ms.apiListNodes)
	engine.GET("/rpc/node/:node/hostfile", ms.apiGetNodeFile)
	engine.POST("/rpc/node/:node/hostfile", ms.apiPushNodeFile)
	return engine
}

func (ms *service) apiPushNodeFile(gctx *gin.Context) {
	hostName := types.CleanString(gctx.Param("node"))
	if hostName == "" {
		gctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	data, err := gctx.GetRawData()
	if err != nil {
		gctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(ms.cfg.Hosts(), hostName), data, 0755)
	if err != nil {
		gctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiServeHostFile(gctx *gin.Context) {
	gctx.File(ms.cfg.HostFile())
}

func (ms *service) apiWatchNode(gctx *gin.Context) {
	var subnet types.Subnet
	if err := gctx.Bind(&subnet); err != nil {
		return
	}
	node := ms.nodes.TryAdd(ms.globalContext, subnet.Node, subnet.Subnet)
	if node != nil {
		ms.events.Connected.emit(node)
		ms.pool.Add(1)
		go func() {
			ms.requestNode(node)
			ms.pool.Done()
		}()
	}
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiForgetNode(gctx *gin.Context) {
	var subnet types.Subnet
	if err := gctx.Bind(&subnet); err != nil {
		return
	}
	node := ms.nodes.TryRemove(subnet.Node)
	if node != nil {
		ms.events.Disconnected.emit(node)
		node.Stop()
	}
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiKillNode(gctx *gin.Context) {
	ms.Stop()
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiListNodes(gctx *gin.Context) {
	var reply NodeList
	reply.Nodes = ms.nodes.Copy()
	gctx.JSON(http.StatusOK, reply)
}

func (ms *service) apiGetNodeFile(gctx *gin.Context) {
	node := gctx.Param("node")
	gctx.File(filepath.Join(ms.cfg.Hosts(), node))
}

func (ms *service) apiUiAddNode(gctx *gin.Context) {
	var params = generator.Config{
		Network: ms.cfg.Network(),
	}
	if err := gctx.Bind(&params); err != nil {
		return
	}
	assembly, err := params.Generate(ms.cfg.Dir)
	if err != nil {
		ms.renderMainPage(gctx, err, "")
		return
	}
	var atLeastOnePublic bool
	for _, node := range ms.nodes.Copy() {
		if err := node.Client().PushNodeFile(params.Name, assembly.PublicKey); err != nil {
			log.Println("UI", err)
		} else if node.Public {
			atLeastOnePublic = true
		}
	}

	if !atLeastOnePublic {
		ms.renderMainPage(gctx, nil, "can't distribute even to one public node")
		return
	}

	gctx.Header("Content-Disposition", "attachment; filename=\""+params.Name+".sh\"")
	gctx.Data(http.StatusOK, "application/bash", assembly.Script)
}

//go:generate go-bindata -o assets.go -pkg monitor --prefix assets/ assets/
func (ms *service) apiUiNodes(gctx *gin.Context) {
	ms.renderMainPage(gctx, nil, "")
}

func (ms *service) renderMainPage(gctx *gin.Context, responseErr error, warn string) {
	list := ms.nodes.Copy()
	var hasPublic bool
	for _, n := range list {
		if n.Public {
			hasPublic = true
			break
		}
	}
	ms.renderTemplate(gctx, "nodes.gotemplate", gin.H{
		"Nodes":     list,
		"Service":   ms,
		"Error":     responseErr,
		"Warning":   warn,
		"HasPublic": hasPublic,
	})
}

func (ms *service) renderTemplate(gctx *gin.Context, name string, params interface{}) {
	ms.initTemplates.Do(func() {
		for _, assetName := range AssetNames() {
			templateNodes = make(map[string]*template.Template)
			templateNodes[assetName] = template.Must(template.New("").Parse(string(MustAsset(assetName))))
		}
	})
	buf := &bytes.Buffer{}
	err := templateNodes[name].Execute(buf, params)
	if err != nil {
		gctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	gctx.Data(http.StatusOK, "text/html", buf.Bytes())
}

var templateNodes map[string]*template.Template
