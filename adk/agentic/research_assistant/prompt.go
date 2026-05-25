/*
 * Copyright 2026 CloudWeGo Authors
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

import "fmt"

func agentInstruction(reportPath string) string {
	return fmt.Sprintf(`You are an autonomous research assistant.

Your task is to prepare a concise, evidence-backed research report for an engineering team evaluating AI agent application patterns.

Work as an agent:
1. Call load_research_brief to get the audience, required report shape, and scoring criteria.
2. Use server-side web_search to verify current public information. Keep searches focused.
3. Call score_evidence for at least two concrete pieces of evidence.
4. Do not directly read the report file. If you need to inspect it, check whether it exists first. Otherwise write the final Markdown report directly with the filesystem middleware tool write_file.
5. After write_file succeeds, stop calling tools and reply briefly.

The report must be saved to:
%s

The Markdown report must include these sections exactly:
## Summary
## Evidence
## Risks
## Recommendation
## Next Steps
## Sources

Keep the report compact: no more than 900 words total, no more than three evidence items, and no more than three source URLs.

Do not claim that you performed purchases, registrations, deployments, or external changes.`, reportPath)
}

func userRequest(reportPath string) string {
	return fmt.Sprintf(`Please prepare a research report for an engineering team evaluating AI agent application patterns.

Focus on:
- tool calling
- server-side tools
- filesystem middleware
- observability of agent runs

Use current public information where useful, score the strongest evidence, keep the report under 900 words, do not directly read the report file unless you have checked that it exists, and use write_file to save the final report to:
%s`, reportPath)
}
