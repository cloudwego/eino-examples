package main

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/document/parser"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	textParser := parser.TextParser{}
	docs, err := textParser.Parse(ctx, strings.NewReader("hello world"))
	if err != nil {
		logs.Errorf("TextParser{}.Parse failed, err=%v", err)
		return
	}

	logs.Infof("text content: %v", docs[0].Content)
}
