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
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/eino-ext/devops"

	"github.com/cloudwego/eino-examples/devops/debug/chain"
	"github.com/cloudwego/eino-examples/devops/debug/graph"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	// init eino devops server
	err := devops.Init(ctx)
	if err != nil {
		logs.Errorf("[eino dev] init failed, err=%v", err)
		return
	}

	// Register chain, graph and state_graph for demo use
	chain.RegisterSimpleChain(ctx)
	graph.RegisterSimpleGraph(ctx)
	graph.RegisterSimpleStateGraph(ctx)

	// This part has nothing to do with eino devops debugging, just wanting the demo service exits only when the user actively closes the process.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// exit
	logs.Infof("[eino dev] shutting down\n")
}
