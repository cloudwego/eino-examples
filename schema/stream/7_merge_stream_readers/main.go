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
	"fmt"
	"io"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

type Event struct {
	Source  string
	Type    string
	Message string
}

func (e Event) String() string {
	return fmt.Sprintf("[%s:%s] %s", e.Source, e.Type, e.Message)
}

func main() {
	logs.Infof("=== MergeStreamReaders Demo ===")
	logs.Infof("Merging events from multiple concurrent sources\n")

	userEvents := createUserEventStream()
	systemMetrics := createSystemMetricsStream()
	notifications := createNotificationStream()

	mergedStream := schema.MergeStreamReaders([]*schema.StreamReader[Event]{
		userEvents,
		systemMetrics,
		notifications,
	})
	defer mergedStream.Close()

	logs.Infof("Receiving merged events (order may vary due to concurrency):\n")

	eventCount := 0
	for {
		event, err := mergedStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Merge error: %v", err)
			break
		}
		eventCount++
		logs.Infof("Event %d: %s", eventCount, event)
	}

	logs.Infof("\nTotal events received: %d", eventCount)
}

func createUserEventStream() *schema.StreamReader[Event] {
	sr, sw := schema.Pipe[Event](1)

	go func() {
		defer sw.Close()
		events := []Event{
			{Source: "user", Type: "login", Message: "User alice logged in"},
			{Source: "user", Type: "action", Message: "User alice viewed dashboard"},
			{Source: "user", Type: "action", Message: "User alice updated profile"},
			{Source: "user", Type: "logout", Message: "User alice logged out"},
		}
		for _, e := range events {
			time.Sleep(80 * time.Millisecond)
			sw.Send(e, nil)
		}
	}()

	return sr
}

func createSystemMetricsStream() *schema.StreamReader[Event] {
	sr, sw := schema.Pipe[Event](1)

	go func() {
		defer sw.Close()
		events := []Event{
			{Source: "system", Type: "cpu", Message: "CPU usage: 45%"},
			{Source: "system", Type: "memory", Message: "Memory usage: 62%"},
			{Source: "system", Type: "disk", Message: "Disk usage: 78%"},
		}
		for _, e := range events {
			time.Sleep(120 * time.Millisecond)
			sw.Send(e, nil)
		}
	}()

	return sr
}

func createNotificationStream() *schema.StreamReader[Event] {
	sr, sw := schema.Pipe[Event](1)

	go func() {
		defer sw.Close()
		events := []Event{
			{Source: "notify", Type: "alert", Message: "High traffic detected"},
			{Source: "notify", Type: "info", Message: "Scheduled maintenance in 2 hours"},
		}
		for _, e := range events {
			time.Sleep(150 * time.Millisecond)
			sw.Send(e, nil)
		}
	}()

	return sr
}
