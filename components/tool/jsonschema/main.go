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

	"github.com/davecgh/go-spew/spew"
	"github.com/eino-contrib/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/cloudwego/eino/schema"
)

func main() {
	JSONSchemaToToolInfo()
}

func JSONSchemaToToolInfo() {
	js := &jsonschema.Schema{
		Type:     string(schema.Object),
		Required: []string{"title"},
		Properties: orderedmap.New[string, *jsonschema.Schema](
			orderedmap.WithInitialData[string, *jsonschema.Schema](
				orderedmap.Pair[string, *jsonschema.Schema]{
					Key: "title",
					Value: &jsonschema.Schema{
						Type: string(schema.String),
					},
				},
				orderedmap.Pair[string, *jsonschema.Schema]{
					Key: "completed",
					Value: &jsonschema.Schema{
						Type: string(schema.Boolean),
					},
				},
			),
		),
	}

	toolInfo := schema.ToolInfo{
		Name:        "todo_manager",
		Desc:        "manage todo list",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(js),
	}

	fmt.Printf("\n=========tool from api path=========\n")
	spew.Dump(toolInfo)
}
