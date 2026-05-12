/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
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
	"flag"
	"fmt"
	"log"
	"strings"
)

func main() {
	step := flag.String("step", "all", "demo step to run: all, plan, search, visual, stateful")
	flag.Parse()

	ctx := context.Background()
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	scenario := newScenario()
	requested := strings.ToLower(strings.TrimSpace(*step))

	switch requested {
	case "all":
		if err = runPlanStep(ctx, cfg, scenario); err != nil {
			log.Fatal(err)
		}
		if err = runSearchStep(ctx, cfg, scenario); err != nil {
			log.Fatal(err)
		}
		if err = runVisualStep(ctx, cfg, scenario); err != nil {
			log.Fatal(err)
		}
		if err = runStatefulStep(ctx, cfg, scenario); err != nil {
			log.Fatal(err)
		}
	case "plan":
		err = runPlanStep(ctx, cfg, scenario)
	case "search":
		seedScenarioForSearch(scenario)
		err = runSearchStep(ctx, cfg, scenario)
	case "visual":
		seedScenarioForVisual(scenario)
		err = runVisualStep(ctx, cfg, scenario)
	case "stateful":
		if err = runPlanStep(ctx, cfg, scenario); err != nil {
			break
		}
		seedScenarioAfterPlanForStateful(scenario)
		err = runStatefulStep(ctx, cfg, scenario)
	default:
		err = fmt.Errorf("unknown step %q, want one of: all, plan, search, visual, stateful", requested)
	}

	if err != nil {
		log.Fatal(err)
	}
}
