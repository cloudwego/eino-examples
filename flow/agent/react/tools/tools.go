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

package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func GetRestaurantTool() tool.InvokableTool {
	return &ToolQueryRestaurants{
		backService: restService,
	}
}

func GetDishTool() tool.InvokableTool {
	return &ToolQueryDishes{
		backService: restService,
	}
}

type ToolQueryRestaurants struct {
	backService *fakeService // fake service
}

func (t *ToolQueryRestaurants) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_restaurants",
		Desc: "Query restaurants",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"location": {
				Type:     "string",
				Desc:     "The location of the restaurant",
				Required: true,
			},
			"topn": {
				Type: "number",
				Desc: "top n restaurant in some location sorted by score",
			},
		}),
	}, nil
}

// InvokableRun
// tool 接收的参数和返回都是 string, 就如大模型的 tool call 的返回一样, 因此需要自行处理参数和结果的序列化.
// 返回的 content 会作为 schema.Message 的 content, 一般来说是作为大模型的输入, 因此处理成大模型能更好理解的结构最好.
// 因此，如果是 json 格式，就需要注意 key 和 value 的表意, 不要用 int Enum 代表一个业务含义，比如 `不要用 1 代表 male, 2 代表 female` 这类.
func (t *ToolQueryRestaurants) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	p := &QueryRestaurantsParam{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		return "", err
	}
	if p.Topn == 0 {
		p.Topn = 3
	}

	// 请求后端服务
	rests, err := t.backService.QueryRestaurants(ctx, p)
	if err != nil {
		return "", err
	}

	// 序列化结果
	res, err := json.Marshal(rests)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

type QueryRestaurantsParam struct {
	Location string `json:"location"`
	Topn     int    `json:"topn"`
}

type Restaurant struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Place string `json:"place"`
	Desc  string `json:"desc"`
	Score int    `json:"score"`
}

// ToolQueryDishes.
type ToolQueryDishes struct {
	backService *fakeService // fake service
}

func (t *ToolQueryDishes) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_dishes",
		Desc: "查询一家餐厅有哪些菜品",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"restaurant_id": {
				Type:     "string",
				Desc:     "The id of one restaurant",
				Required: true,
			},
			"topn": {
				Type: "number",
				Desc: "top n dishes in one restaurant sorted by score",
			},
		}),
	}, nil
}

func (t *ToolQueryDishes) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	p := &QueryDishesParam{}
	err := json.Unmarshal([]byte(argumentsInJSON), p)
	if err != nil {
		return "", err
	}

	if p.Topn == 0 {
		p.Topn = 5
	}

	// 请求后端服务
	rests, err := t.backService.QueryDishes(ctx, p)
	if err != nil {
		return "", err
	}

	// 序列化结果
	res, err := json.Marshal(rests)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

type QueryDishesParam struct {
	RestaurantID string `json:"restaurant_id"`
	Topn         int    `json:"topn"`
}

type Dish struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Price int    `json:"price"`
	Score int    `json:"score"`
}
