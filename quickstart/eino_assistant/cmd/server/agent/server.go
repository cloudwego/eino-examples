package agent

import (
	"bufio"
	"context"
	"embed"
	"errors"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/hertz-contrib/sse"

	"github.com/cloudwego/eino-examples/agent"
	"github.com/cloudwego/eino-examples/agent/mem"
)

//go:embed web
var webContent embed.FS

type ChatRequest struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func BindRoutes(r *route.RouterGroup) error {
	if err := agent.Init(); err != nil {
		return err
	}

	// API 路由
	r.GET("/api/chat", HandleChat)
	r.GET("/api/log", HandleLog)
	r.GET("/api/history", HandleHistory)
	r.DELETE("/api/history", HandleDeleteHistory)

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

func HandleChat(ctx context.Context, c *app.RequestContext) {
	id := c.Query("id")
	message := c.Query("message")
	if id == "" || message == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "missing id or message parameter",
		})
		return
	}

	log.Printf("[Chat] Starting chat with ID: %s, Message: %s\n", id, message)

	sr, err := agent.RunAgent(ctx, id, message)
	if err != nil {
		log.Printf("[Chat] Error running agent: %v\n", err)
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	s := sse.NewStream(c)
	defer func() {
		sr.Close()
		c.Flush()

		log.Printf("[Chat] Finished chat with ID: %s\n", id)
	}()

outter:
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Chat] Context done for chat ID: %s\n", id)
			return
		default:
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				log.Printf("[Chat] EOF received for chat ID: %s\n", id)
				break outter
			}
			if err != nil {
				log.Printf("[Chat] Error receiving message: %v\n", err)
				break outter
			}

			err = s.Publish(&sse.Event{
				Data: []byte(msg.Content),
			})
			if err != nil {
				log.Printf("[Chat] Error publishing message: %v\n", err)
				break outter
			}
		}
	}
}

func HandleHistory(ctx context.Context, c *app.RequestContext) {
	// query: id => get history, none => list all
	id := c.Query("id")

	if id == "" {
		ids := mem.GetDefaultMemory().ListConversations()

		c.JSON(consts.StatusOK, map[string]interface{}{
			"ids": ids,
		})
		return
	}

	conversation := mem.GetDefaultMemory().GetConversation(id, false)
	if conversation == nil {
		c.JSON(consts.StatusNotFound, map[string]string{
			"error": "conversation not found",
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"conversation": conversation,
	})

}

func HandleDeleteHistory(ctx context.Context, c *app.RequestContext) {
	id := c.Query("id")
	if id == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error": "missing id parameter",
		})
		return
	}

	mem.GetDefaultMemory().DeleteConversation(id)
	c.JSON(consts.StatusOK, map[string]string{
		"status": "success",
	})
}

func HandleLog(ctx context.Context, c *app.RequestContext) {
	file, err := os.Open("log/eino.log")
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	defer file.Close()

	// Create a new SSE stream
	s := sse.NewStream(c)
	defer c.Flush()

	// Seek to the end of the file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		log.Println("error seeking file:", err)
		return
	}

	// Use a goroutine to continuously read new lines
	go func() {
		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				log.Println("error reading log:", err)
				break
			}

			// If we got a line, publish it
			if line != "" {
				err = s.Publish(&sse.Event{
					Data: []byte(line),
				})
				if err != nil {
					log.Println("error publishing log:", err)
					break
				}
			}

			// If we hit EOF, wait a bit and try again
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
	}()

	// Keep the connection open
	<-ctx.Done()
}
