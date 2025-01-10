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
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/cloudwego/eino/schema"
)

func main() {
	OpenapiDocToToolInfo()
}

func OpenapiDocToToolInfo() {
	loader := openapi3.NewLoader()

	// 解析 JSON 文档为 json Schema
	doc, err := loader.LoadFromFile("./openapi.json")
	if err != nil {
		log.Fatalf("解析 openapi.json 失败: %v", err)
	}

	// **** 如果是对 path 的 body 引用 ****
	// eg: POST /api/v1/todo
	schemaVal01, err := GetSchemaFromPath(doc, "POST", "/api/v1/todo")
	if err != nil {
		log.Fatalf("获取 ref 失败: %v", err)
	}

	toolInfo := schema.ToolInfo{
		Name:        "todo_manager",
		Desc:        "manage todo list",
		ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(schemaVal01),
	}

	fmt.Printf("\n=========tool from api path=========\n")
	spew.Dump(toolInfo)

	// **** 如果有 ref 引用 ****
	// eg: refName "#/components/schemas/TodoRequest" => ref := "TodoRequest"
	schemaVal02, err := GetSchemaFromRef(doc, "TodoRequest")
	if err != nil {
		log.Fatalf("获取 ref 失败: %v", err)
	}
	testToolInfo := schema.ToolInfo{
		Name:        "test",
		Desc:        "test desc",
		ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(schemaVal02),
	}

	fmt.Printf("\n\n=========tool from schema ref=========\n")
	spew.Dump(testToolInfo)
}

// 获取引用的 schema
func GetSchemaFromRef(doc *openapi3.T, ref string) (*openapi3.Schema, error) {
	schemaRef, ok := doc.Components.Schemas[ref]
	if !ok {
		return nil, fmt.Errorf("未找到引用: %s", ref)
	}

	return schemaRef.Value, nil
}

// 获取 path 的 schema
func GetSchemaFromPath(doc *openapi3.T, method string, path string) (*openapi3.Schema, error) {
	pattItem := doc.Paths.Find(path)
	if pattItem == nil {
		return nil, fmt.Errorf("未找到 path: %s", path)
	}

	methodItem := pattItem.GetOperation(method)
	if methodItem == nil {
		return nil, fmt.Errorf("未找到 method: %s", method)
	}

	reqBody := methodItem.RequestBody
	if reqBody == nil || reqBody.Value == nil || reqBody.Value.Content == nil {
		return nil, fmt.Errorf("未找到 requestBody: %s %s", method, path)
	}

	jschema := reqBody.Value.Content["application/json"]
	if jschema == nil || jschema.Schema == nil {
		return nil, fmt.Errorf("未找到 schema: %s %s", method, path)
	}

	return jschema.Schema.Value, nil
}
