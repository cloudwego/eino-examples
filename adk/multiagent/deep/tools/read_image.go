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

package tools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	jsoniter "github.com/json-iterator/go"
)

var (
	toolImageReaderInfo = &schema.ToolInfo{
		Name: "image_reader",
		Desc: "Tool for describing image content",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     "string",
				Desc:     "Questions posed about the image",
				Required: true,
			},
			"image_path": {
				Type:     "string",
				Desc:     "The path of the image file",
				Required: true,
			},
		}),
	}
)

func NewToolImageReader(visionModel model.BaseChatModel) tool.InvokableTool {
	return &localToolImageReader{visionModel: visionModel}
}

type localToolImageReader struct {
	visionModel model.BaseChatModel
}

func (t *localToolImageReader) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolImageReaderInfo, nil
}

func (t *localToolImageReader) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var params struct {
		Query     string `json:"query"`
		ImagePath string `json:"image_path"`
	}
	if err := jsoniter.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", err
	}
	if params.Query == "" || params.ImagePath == "" {
		return "", errors.New("missing parameters")
	}

	f, err := os.Open(params.ImagePath)
	if err != nil {
		return fmt.Sprintf("open file error: %v, file path: %v", err, params.ImagePath), nil
	}
	defer f.Close()
	fc, err := io.ReadAll(f)
	if err != nil {
		return fmt.Sprintf("read file error: %v, file path: %v", err, params.ImagePath), nil
	}

	mimeType := http.DetectContentType(fc)
	b64 := base64.StdEncoding.EncodeToString(fc)
	url := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
	msgs := []*schema.Message{
		schema.SystemMessage(""), // TODO: fill system prompt
		schema.UserMessage(params.Query),
		{
			Role: schema.User,
			UserInputMultiContent: []schema.MessageInputPart{
				{
					Type: schema.ChatMessagePartTypeImageURL,
					Image: &schema.MessageInputImage{
						MessagePartCommon: schema.MessagePartCommon{
							URL:      &url,
							MIMEType: mimeType,
						},
						Detail: "",
					},
				},
			},
		},
	}

	resp, err := t.visionModel.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}

	if resp.Content == "" {
		return "", errors.New("response is empty")
	}

	return resp.Content, nil
}
