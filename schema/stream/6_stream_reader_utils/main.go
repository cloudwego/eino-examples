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
	"strconv"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	logs.Infof("=== StreamReader Utilities Demo ===\n")

	demoPipe(ctx)
	demoStreamReaderFromArray()
	demoStreamReaderWithConvert()
	demoErrNoValue()
}

func demoPipe(ctx context.Context) {
	logs.Infof("--- 1. schema.Pipe[T] ---")
	logs.Infof("Creates a StreamReader/StreamWriter pair for async data production\n")

	sr, sw := schema.Pipe[string](2)

	go func() {
		defer sw.Close()
		for i := 1; i <= 5; i++ {
			time.Sleep(100 * time.Millisecond)
			sw.Send("chunk-"+strconv.Itoa(i), nil)
		}
	}()

	logs.Infof("Receiving from Pipe:")
	for {
		chunk, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Pipe error: %v", err)
			break
		}
		logs.Tokenf("[%s] ", chunk)
	}
	logs.Infof("\n")
}

func demoStreamReaderFromArray() {
	logs.Infof("--- 2. schema.StreamReaderFromArray ---")
	logs.Infof("Converts an array/slice into a StreamReader\n")

	data := []string{"apple", "banana", "cherry", "date"}
	sr := schema.StreamReaderFromArray(data)
	defer sr.Close()

	logs.Infof("Reading from array stream:")
	for {
		item, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Array stream error: %v", err)
			break
		}
		logs.Tokenf("[%s] ", item)
	}
	logs.Infof("\n")
}

func demoStreamReaderWithConvert() {
	logs.Infof("--- 3. schema.StreamReaderWithConvert ---")
	logs.Infof("Transforms stream elements from one type to another\n")

	numbers := []string{"1", "2", "3", "4", "5"}
	strStream := schema.StreamReaderFromArray(numbers)

	intStream := schema.StreamReaderWithConvert(strStream, func(s string) (int, error) {
		return strconv.Atoi(s)
	})
	defer intStream.Close()

	logs.Infof("Converting string stream to int stream:")
	sum := 0
	for {
		num, err := intStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Convert error: %v", err)
			break
		}
		logs.Tokenf("[%d] ", num)
		sum += num
	}
	logs.Infof("\nSum: %d\n", sum)
}

func demoErrNoValue() {
	logs.Infof("--- 4. schema.ErrNoValue ---")
	logs.Infof("Filters out elements from a stream by returning ErrNoValue\n")

	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	numStream := schema.StreamReaderFromArray(numbers)

	evenStream := schema.StreamReaderWithConvert(numStream, func(n int) (int, error) {
		if n%2 != 0 {
			return 0, schema.ErrNoValue
		}
		return n, nil
	})
	defer evenStream.Close()

	logs.Infof("Filtering for even numbers only:")
	for {
		num, err := evenStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Filter error: %v", err)
			break
		}
		logs.Tokenf("[%d] ", num)
	}
	logs.Infof("\n")
}
