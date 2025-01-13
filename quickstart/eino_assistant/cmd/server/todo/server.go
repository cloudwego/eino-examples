package todo

import (
	"context"
	"embed"
	"mime"
	"path/filepath"

	"github.com/cloudwego/eino-examples/agent/tool/todo"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

//go:embed web
var webContent embed.FS

// BindRoutes 注册路由
func BindRoutes(r *route.RouterGroup) error {
	todoTool, err := todo.NewTodoTool(todo.GetDefaultStorage())
	if err != nil {
		return err
	}

	// API 处理
	r.POST("/api", func(c context.Context, ctx *app.RequestContext) {
		var req todo.TodoRequest
		if err := ctx.Bind(&req); err != nil {
			ctx.JSON(consts.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		resp, err := todoTool.Invoke(c, &req)
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		ctx.JSON(consts.StatusOK, resp)
	})

	// 静态文件服务
	r.GET("/", func(c context.Context, ctx *app.RequestContext) {
		content, err := webContent.ReadFile("web/index.html")
		if err != nil {
			ctx.String(consts.StatusNotFound, "File not found")
			return
		}
		ctx.Header("Content-Type", "text/html")
		ctx.Write(content)
	})

	r.GET("/:file", func(c context.Context, ctx *app.RequestContext) {
		file := ctx.Param("file")
		content, err := webContent.ReadFile("web/" + file)
		if err != nil {
			ctx.String(consts.StatusNotFound, "File not found")
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(file))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		ctx.Header("Content-Type", contentType)
		ctx.Write(content)
	})

	return nil
}
