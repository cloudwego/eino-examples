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

package logs

import (
	"fmt"
	"os"
	"time"
)

const (
	// Color codes for terminal output
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorBrown = "\033[31;1m"
	colorReset = "\033[0m"
)

func Infof(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("%s[INFO] %s ", colorGreen, timestamp)
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", prefix, message, colorReset)
}

func Errorf(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("%s[ERROR] %s ", colorRed, timestamp)
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", prefix, message, colorReset)
}

func Tokenf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s", colorBrown, message, colorReset)
}

func Fatalf(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("%s[FATAL] %s ", colorRed, timestamp)
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", prefix, message, colorReset)
	os.Exit(1)
}
