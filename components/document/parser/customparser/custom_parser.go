/*
 * Copyright 2025 CloudWeGo Authors
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
	"io"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

// options
// 定制实现自主定义的 option 结构体
type options struct {
	Encoding string
	MaxSize  int64
}

// WithEncoding
// 定制实现自主定义的 Option 方法
func WithEncoding(encoding string) parser.Option {
	return parser.WrapImplSpecificOptFn(func(o *options) {
		o.Encoding = encoding
	})
}

func WithMaxSize(size int64) parser.Option {
	return parser.WrapImplSpecificOptFn(func(o *options) {
		o.MaxSize = size
	})
}

type Config struct {
	DefaultEncoding string
	DefaultMaxSize  int64
}

type CustomParser struct {
	defaultEncoding string
	defaultMaxSize  int64
}

func NewCustomParser(config *Config) (*CustomParser, error) {
	return &CustomParser{
		defaultEncoding: config.DefaultEncoding,
		defaultMaxSize:  config.DefaultMaxSize,
	}, nil
}

func (p *CustomParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	// 1. 处理通用选项
	commonOpts := parser.GetCommonOptions(&parser.Options{}, opts...)
	_ = commonOpts

	// 2. 处理特定选项
	myOpts := &options{
		Encoding: p.defaultEncoding,
		MaxSize:  p.defaultMaxSize,
	}
	myOpts = parser.GetImplSpecificOptions(myOpts, opts...)
	_ = myOpts
	// 3. 实现解析逻辑

	return []*schema.Document{{
		Content: "Hello World",
	}}, nil
}
