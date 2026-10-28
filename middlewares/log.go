package middlewares

import (
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/gin-gonic/gin"
)

// RequestLog 在读取正文前记录请求到达，省略查询参数、正文和请求头
func RequestLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		global.Logger.Infof("收到 HTTP 登录请求：method:%s path:%q", ctx.Request.Method, ctx.Request.URL.Path)
		ctx.Next()
	}
}
