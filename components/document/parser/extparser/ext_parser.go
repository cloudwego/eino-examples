/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/document/parser/html"
	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document/parser"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"

	"github.com/xuri/excelize/v2"
)

// xlsxParser 自定义解析器，用于解析excel文件内容
type xlsxParserImpl struct {
	config *xlsxParserConfig
}

// xlsxParserConfig 用于配置xlsxParser
type xlsxParserConfig struct {
	SheetName string // 指定要处理的工作表名称，为空则处理所有工作表
	HasHeader bool   // 是否包含表头
}

// newXlsxParser 创建一个新的xlsxParser
func newXlsxParser(ctx context.Context, hasHeader bool) (xlp parser.Parser, err error) {
	// 配置HasHeader为true，表示第一行为表头
	config := &xlsxParserConfig{
		HasHeader: hasHeader,
		// SheetName: "sheet1",
	}
	xlp = &xlsxParserImpl{config: config}
	return xlp, nil
}

// Parse 实现自定义解析器接口
func (impl xlsxParserImpl) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	xlFile, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, err
	}
	defer xlFile.Close()

	// 获取所有工作表
	sheets := xlFile.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil
	}

	// 确定要处理的工作表，默认只处理第一个工作表
	sheetName := sheets[0]
	if impl.config.SheetName != "" {
		sheetName = impl.config.SheetName
	}

	// 获取所有行，表头+数据行
	rows, err := xlFile.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	var ret []*schema.Document

	// 处理表头
	startIdx := 0
	var headers []string
	if impl.config.HasHeader && len(rows) > 0 {
		headers = rows[0]
		startIdx = 1
	}
	// 处理数据行
	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// 将行数据转换为字符串
		contentParts := make([]string, len(row))
		for j, cell := range row {
			contentParts[j] = strings.TrimSpace(cell)
		}
		content := strings.Join(contentParts, "\t")

		// 创建新的Document
		nDoc := &schema.Document{
			ID:       fmt.Sprintf("%d", i),
			Content:  fmt.Sprintf("%s", content),
			MetaData: map[string]any{},
		}

		// 如果有表头，将数据添加到元数据中
		if impl.config.HasHeader {
			if nDoc.MetaData == nil {
				nDoc.MetaData = make(map[string]any)
			}
			for j, header := range headers {
				if j < len(row) {
					nDoc.MetaData[header] = row[j]
				}
			}
		}

		ret = append(ret, nDoc)
	}

	return ret, nil
}

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
	xlsxParser, err := newXlsxParser(ctx, true)
	if err != nil {
		logs.Errorf("newXlsxParser failed, err=%v", err)
		return
	}
	// 创建扩展解析器
	extParser, err := parser.NewExtParser(ctx, &parser.ExtParserConfig{
		// 注册特定扩展名的解析器
		Parsers: map[string]parser.Parser{
			".html": htmlParser,
			".pdf":  pdfParser,
			".xlsx": xlsxParser,
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
	if err != nil {
		logs.Errorf("extParser.Parse, err=%v", err)
		return
	}

	for idx, doc := range docs {
		logs.Infof("doc_%v content: %v metedata: %v", idx, doc.Content, doc.MetaData)
	}
}
