package router

import (
	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/controllers"
	"github.com/GXJY-Project/gxjy-server/internal/server"
	"github.com/GXJY-Project/gxjy-server/middlewares"
	"github.com/gin-gonic/gin"
)

// SetupRouter 注册 HTTP 登录入口，参数来自统一运行配置
func SetupRouter() *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middlewares.RequestLog())
	login := controllers.LoginHandler{
		GateHost: config.AppConfig.GateHost,
		HTTPPort: config.AppConfig.HTTPPort,
		WSPort:   config.AppConfig.WSPort,
		Token:    "local-token-0001",
	}
	r.Any("/healthz", login.Handle)
	r.Any("/version.manifest", login.Handle)
	r.Any("/project.manifest", login.Handle)
	r.Any("/getResource", login.Handle)
	player := r.Group("/player")
	{
		player.Any("/login", login.Handle)
		player.Any("/getResource", login.Handle)
		player.Any("/enter", login.Handle)
	}
	// Python make_login_response 也处理其他路径，保留该回退且不占用通配路由
	r.NoRoute(login.Handle)
	return r
}

// SetupGameRouter 注册独立端口的 WebSocket 入口
func SetupGameRouter(app *server.App) *gin.Engine {
	r := gin.New()
	// 保留 /game 的精确路径匹配，不将 /game/ 重定向为有效入口
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	handler := gin.WrapH(app.GameHandler())
	r.Any("/game", handler)
	r.NoRoute(handler)
	return r
}
