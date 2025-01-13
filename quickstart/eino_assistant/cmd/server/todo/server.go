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
	ctx := context.Background()

	todoTool, err := todo.NewTodoToolImpl(ctx, &todo.TodoToolConfig{
		Storage: todo.GetDefaultStorage(),
	})
	if err != nil {
		return err
	}

	// API 处理
	r.POST("/api", func(ctx context.Context, c *app.RequestContext) {
		var req todo.TodoRequest
		if err := c.Bind(&req); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		resp, err := todoTool.Invoke(ctx, &req)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(consts.StatusOK, resp)
	})

	// 静态文件服务
	r.GET("/", func(ctx context.Context, c *app.RequestContext) {
		content, err := webContent.ReadFile("web/index.html")
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		c.Header("Content-Type", "text/html")
		c.Write(content)
	})

	r.GET("/:file", func(ctx context.Context, c *app.RequestContext) {
		file := c.Param("file")
		content, err := webContent.ReadFile("web/" + file)
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(file))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Header("Content-Type", contentType)
		c.Write(content)
	})

	return nil
}
