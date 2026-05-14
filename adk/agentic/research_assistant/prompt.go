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
4. Write a Markdown report and call save_research_report with the full markdown content.
5. If save_research_report returns blocked_by_evidence_gate, revise the report and call save_research_report again.
6. After the report is saved, stop calling tools and reply briefly.

The report must be saved to:
%s

The Markdown report must include these sections exactly:
## Summary
## Evidence
## Risks
## Recommendation
## Next Steps
## Sources

Do not claim that you performed purchases, registrations, deployments, or external changes.`, reportPath)
}

func userRequest(reportPath string) string {
	return fmt.Sprintf(`Please prepare a research report for an engineering team evaluating AI agent application patterns.

Focus on:
- tool calling
- server-side tools
- middleware or policy gates
- observability of agent runs

Use current public information where useful, score the strongest evidence, and save the final report to:
%s`, reportPath)
}
