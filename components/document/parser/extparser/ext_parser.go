package main

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/document/parser/html"
	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document/parser"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	textParser := parser.TextParser{}

	htmlParser, err := html.NewParser(ctx, &html.Config{
		Selector: gptr.Of("body"),
	})
	if err != nil {
		logs.Errorf("html.NewParser failed, err=%v", err)
		return
	}

	pdfParser, err := pdf.NewPDFParser(ctx, &pdf.Config{})
	if err != nil {
		logs.Errorf("pdf.NewPDFParser failed, err=%v", err)
		return
	}

	// 创建扩展解析器
	extParser, err := parser.NewExtParser(ctx, &parser.ExtParserConfig{
		// 注册特定扩展名的解析器
		Parsers: map[string]parser.Parser{
			".html": htmlParser,
			".pdf":  pdfParser,
		},
		// 设置默认解析器，用于处理未知格式
		FallbackParser: textParser,
	})
	if err != nil {

		return
	}

	// 使用解析器
	filePath := "./testdata/test.html"
	file, err := os.Open(filePath)
	if err != nil {
		logs.Errorf("os.Open failed, file=%v, err=%v", filePath, err)
		return
	}
	docs, err := extParser.Parse(ctx, file,
		// 必须提供 URI ExtParser 选择正确的解析器进行解析
		parser.WithURI(filePath),
		parser.WithExtraMeta(map[string]any{
			"source": "local",
		}),
	)

	for idx, doc := range docs {
		logs.Infof("doc_%v content: %v", idx, doc.Content)
	}
}
