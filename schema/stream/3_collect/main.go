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
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

type LogStats struct {
	TotalCount   int
	ErrorCount   int
	WarningCount int
	InfoCount    int
}

func (s LogStats) String() string {
	return fmt.Sprintf("Total: %d, Errors: %d, Warnings: %d, Info: %d",
		s.TotalCount, s.ErrorCount, s.WarningCount, s.InfoCount)
}

func main() {
	ctx := context.Background()

	logAggregator := compose.CollectableLambda(func(ctx context.Context, logStream *schema.StreamReader[string]) (*LogStats, error) {
		logs.Infof("Starting log aggregation...")

		stats := &LogStats{}

		for {
			entry, err := logStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read log entry: %w", err)
			}

			stats.TotalCount++
			upperEntry := strings.ToUpper(entry)

			switch {
			case strings.Contains(upperEntry, "ERROR"):
				stats.ErrorCount++
				logs.Errorf("Log entry: %s", entry)
			case strings.Contains(upperEntry, "WARN"):
				stats.WarningCount++
				logs.Infof("Log entry: %s", entry)
			default:
				stats.InfoCount++
				logs.Infof("Log entry: %s", entry)
			}
		}

		logs.Infof("Aggregation complete: %s", stats)
		return stats, nil
	})

	graph := compose.NewGraph[string, *LogStats]()
	_ = graph.AddLambdaNode("aggregator", logAggregator)
	_ = graph.AddEdge(compose.START, "aggregator")
	_ = graph.AddEdge("aggregator", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Failed to compile graph: %v", err)
	}

	logEntries := []string{
		"[INFO] Application started successfully",
		"[INFO] Connected to database",
		"[WARN] High memory usage detected",
		"[ERROR] Failed to process request: timeout",
		"[INFO] Request completed in 150ms",
		"[WARN] Slow query detected: 2.5s",
		"[ERROR] Connection lost to external service",
		"[INFO] Reconnection successful",
		"[INFO] Processing batch job",
		"[ERROR] Batch job failed: invalid data",
	}

	logStream := schema.StreamReaderFromArray(logEntries)

	logs.Infof("Processing %d log entries...", len(logEntries))

	stats, err := runner.Collect(ctx, logStream)
	if err != nil {
		logs.Fatalf("Failed to aggregate logs: %v", err)
	}

	logs.Infof("Final statistics: %s", stats)
}
