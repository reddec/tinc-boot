package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/reddec/tinc-boot/types"
	"net/http"
)

func (ms *service) createAPI() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/", ms.apiServeHostFile)
	engine.POST("/rpc/watch", ms.apiWatchNode)
	engine.POST("/rpc/forget", ms.apiForgetNode)
	engine.POST("/rpc/kill", ms.apiKillNode)

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
