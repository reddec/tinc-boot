package monitor

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/reddec/tinc-boot/types"
	"html/template"
	"net/http"
	"path/filepath"
)

func (ms *service) createAPI() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/", ms.apiServeHostFile)
	engine.GET("/ui", ms.apiUiNodes)
	engine.POST("/rpc/watch", ms.apiWatchNode)
	engine.POST("/rpc/forget", ms.apiForgetNode)
	engine.POST("/rpc/kill", ms.apiKillNode)
	engine.GET("/rpc/nodes", ms.apiListNodes)
	engine.GET("/rpc/node/:node/hostfile", ms.apiGetNodeFile)
	return engine
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
		node.Stop()
	}
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiKillNode(gctx *gin.Context) {
	ms.Stop()
	gctx.AbortWithStatus(http.StatusNoContent)
}

func (ms *service) apiListNodes(gctx *gin.Context) {
	var reply struct {
		Nodes []*Node `json:"nodes"`
	}
	reply.Nodes = ms.nodes.Copy()
	gctx.JSON(http.StatusOK, reply)
}

func (ms *service) apiGetNodeFile(gctx *gin.Context) {
	node := gctx.Param("node")
	gctx.File(filepath.Join(ms.cfg.Hosts(), node))
}

//go:generate go-bindata -o assets.go -pkg monitor --prefix assets/ assets/
func (ms *service) apiUiNodes(gctx *gin.Context) {
	nodes := ms.nodes.Copy()
	ms.renderTemplate(gctx, "nodes.gotemplate", gin.H{
		"Nodes":   nodes,
		"Service": ms,
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
