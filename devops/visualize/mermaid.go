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

package visualize

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/cloudwego/eino/compose"
)

// MermaidGenerator renders a Mermaid diagram from a compiled Eino graph (Graph/Chain/Workflow).
//
// Core concepts and mapping:
// - Nodes: labeled with their key and component type. Lambda nodes use rounded shapes.
// - Special nodes: START/END are rendered with safe IDs (start_node/end_node) to avoid Mermaid keyword conflicts.
// - SubGraphs: nested Graph/Chain/Workflow are rendered as Mermaid sub-graphs with their component type in the title.
// - Edges:
//   - In general graphs/chains: a single solid arrow (-->), representing standard control+data execution.
//   - In workflows (workflowStyle=true): edges are distinguished by semantics:
//   - control+data:  normal arrow with label "control+data" ("-- control+data -->")
//   - control-only:  bold arrow with label "control-only" ("== control-only ==>")
//   - data-only:     dotted arrow with label "data-only" ("-. data-only .->")
//     Branch decision diamonds and their incoming/outgoing edges are treated as control-only in workflows.
//
// Usage:
//
//	buf := &bytes.Buffer{}
//	gen := visualize.NewMermaidGenerator(buf)                // for Graph/Chain
//	// or
//	gen := visualize.NewMermaidGeneratorWorkflow(buf)        // for Workflow with labeled edges
//	_, _ = g.Compile(ctx, compose.WithGraphCompileCallbacks(gen), compose.WithGraphName("MyGraph"))
//	// Write to a Markdown file:
//	md := "```mermaid\n" + buf.String() + "\n```\n"
//	_ = os.WriteFile("my_graph.md", []byte(md), 0644)
type MermaidGenerator struct {
	w             io.Writer
	workflowStyle bool
	autoWrite     bool
	outDir        string
	baseName      string
	makeImages    bool
}

// NewMermaidGenerator creates a generator that auto-writes Markdown and attempts PNG/SVG generation.
// If dir is empty, current working directory is used. File name is derived from graph name or defaults to "topology".
func NewMermaidGenerator(dir string) *MermaidGenerator {
	return &MermaidGenerator{autoWrite: true, outDir: dir, makeImages: true}
}

// OnFinish is the compile callback entrypoint invoked by Eino after graph compilation.
// It reads the compile-time GraphInfo and writes a complete Mermaid diagram to the writer.
func (m *MermaidGenerator) OnFinish(_ context.Context, info *compose.GraphInfo) {
	m.generate(info)
}

// generate orchestrates diagram construction by delegating to renderGraph.
// The top-level direction is TD (top-down) for readability and consistency.
func (m *MermaidGenerator) generate(info *compose.GraphInfo) {
	var isWorkflow bool
	if len(info.Edges) > len(info.DataEdges) {
		isWorkflow = true
	}

	if !isWorkflow {
		for from, edges := range info.Edges {
			dataEdges, ok := info.DataEdges[from]
			if !ok {
				isWorkflow = true
				break
			}

			if len(edges) != len(dataEdges) {
				isWorkflow = true
				break
			}

			for i := range edges {
				edge := edges[i]
				found := false
				for _, dEdge := range dataEdges {
					if dEdge == edge {
						found = true
						break
					}
				}
				if !found {
					isWorkflow = true
					break
				}
			}
		}
	}

	sb := &strings.Builder{}
	sb.WriteString("graph TD\n")
	m.renderGraph(sb, info, "", 1, isWorkflow)
	if m.w != nil && !m.autoWrite {
		_, _ = fmt.Fprint(m.w, sb.String())
		return
	}

	dir := m.outDir
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		} else {
			dir = "."
		}
	}
	name := m.baseName
	if name == "" {
		if len(info.Name) > 0 {
			name = sanitize(info.Name)
		} else {
			name = "topology"
		}
	}
	mdPath := filepath.Join(dir, name+".md")
	content := sb.String()
	_ = os.WriteFile(mdPath, []byte("```mermaid\n"+content+"\n```"), 0644)
	if m.makeImages {
		mmdPath := filepath.Join(dir, name+".mmd")
		_ = os.WriteFile(mmdPath, []byte(content), 0644)
		m.renderImage(mmdPath, filepath.Join(dir, name+".png"))
		_ = os.Remove(mmdPath)
	}
}

// renderGraph builds a Mermaid diagram section for the given GraphInfo.
// It:
// 1) Collects and sorts node keys for deterministic output
// 2) Renders nodes (including nested sub-graphs)
// 3) Renders control/data edges with workflow-aware styles
// 4) Renders branches with decision diamonds and proper edge types
func (m *MermaidGenerator) renderGraph(sb *strings.Builder, info *compose.GraphInfo, prefix string, indentLevel int, style bool) {
	indent := strings.Repeat("  ", indentLevel)

	// Collect all nodes from info.Nodes, Edges, and Branches
	allNodes := make(map[string]bool)
	for k := range info.Nodes {
		allNodes[k] = true
	}
	for start, ends := range info.Edges {
		allNodes[start] = true
		for _, end := range ends {
			allNodes[end] = true
		}
	}
	for start, branches := range info.Branches {
		allNodes[start] = true
		for _, branch := range branches {
			endNodes := branch.GetEndNode()
			for end := range endNodes {
				allNodes[end] = true
			}
		}
	}

	// Sort nodes for deterministic output
	nodes := make([]string, 0, len(allNodes))
	for k := range allNodes {
		nodes = append(nodes, k)
	}
	sort.Strings(nodes)

	// Render Nodes
	for _, nodeKey := range nodes {
		nodeID := m.nodeID(prefix, nodeKey)

		if nodeInfo, ok := info.Nodes[nodeKey]; ok {
			if nodeInfo.GraphInfo != nil {
				// Subgraph
				subgraphLabel := nodeKey
				if nodeInfo.Component == compose.ComponentOfChain {
					subgraphLabel = fmt.Sprintf("%s (Chain)", nodeKey)
				} else if nodeInfo.Component == compose.ComponentOfWorkflow {
					subgraphLabel = fmt.Sprintf("%s (Workflow)", nodeKey)
				} else if nodeInfo.Component == compose.ComponentOfGraph {
					subgraphLabel = fmt.Sprintf("%s (Graph)", nodeKey)
				}
				sb.WriteString(fmt.Sprintf("%ssubgraph %s [\"%s\"]\n", indent, nodeID, subgraphLabel))
				childStyle := style
				if nodeInfo.Component == compose.ComponentOfWorkflow {
					childStyle = true
				} else if nodeInfo.Component == compose.ComponentOfGraph || nodeInfo.Component == compose.ComponentOfChain {
					// for explicit Graph/Chain sub-graphs, do not apply workflow styling
					childStyle = false
				}
				m.renderGraph(sb, nodeInfo.GraphInfo, nodeID+"_", indentLevel+1, childStyle)
				sb.WriteString(fmt.Sprintf("%send\n", indent))
			} else {
				// Regular Node
				shapeStart, shapeEnd := "[", "]"
				if nodeInfo.Component == compose.ComponentOfLambda {
					shapeStart, shapeEnd = "(", ")"
				}

				label := fmt.Sprintf("%s<br/>(%s)", nodeKey, nodeInfo.Component)
				sb.WriteString(fmt.Sprintf("%s%s%s\"%s\"%s\n", indent, nodeID, shapeStart, label, shapeEnd))
			}
		} else if nodeKey == compose.START || nodeKey == compose.END {
			// Special nodes: avoid reserved keyword conflict with 'end'
			safeID := nodeID
			if nodeKey == compose.START {
				safeID = m.nodeID(prefix, "start_node")
			} else {
				safeID = m.nodeID(prefix, "end_node")
			}
			sb.WriteString(fmt.Sprintf("%s%s([%s])\n", indent, safeID, nodeKey))
		}
	}

	// Render Control Edges
	// Sort edges for deterministic output
	startNodes := make([]string, 0, len(info.Edges))
	for k := range info.Edges {
		startNodes = append(startNodes, k)
	}
	sort.Strings(startNodes)

	for _, start := range startNodes {
		ends := info.Edges[start]
		for _, end := range ends {
			startID := m.nodeID(prefix, start)
			endID := m.nodeID(prefix, end)
			if start == compose.START {
				startID = m.nodeID(prefix, "start_node")
			}
			if end == compose.END {
				endID = m.nodeID(prefix, "end_node")
			}
			// Determine edge semantics by checking if a matching data edge exists.
			hasData := false
			if des, ok := info.DataEdges[start]; ok {
				for _, de := range des {
					if de == end {
						hasData = true
						break
					}
				}
			}
			if style {
				if hasData {
					sb.WriteString(fmt.Sprintf("%s%s -- control+data --> %s\n", indent, startID, endID))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s == control-only ==> %s\n", indent, startID, endID))
				}
			} else {
				sb.WriteString(fmt.Sprintf("%s%s --> %s\n", indent, startID, endID))
			}
		}
	}

	// Render Data Edges
	// Only render if they differ from control edges; otherwise already represented as control+data.
	dataStartNodes := make([]string, 0, len(info.DataEdges))
	for k := range info.DataEdges {
		dataStartNodes = append(dataStartNodes, k)
	}
	sort.Strings(dataStartNodes)

	for _, start := range dataStartNodes {
		ends := info.DataEdges[start]
		for _, end := range ends {
			// Check if this edge already exists as a control edge
			alreadyExists := false
			for _, controlEnd := range info.Edges[start] {
				if controlEnd == end {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				startID := m.nodeID(prefix, start)
				endID := m.nodeID(prefix, end)
				if start == compose.START {
					startID = m.nodeID(prefix, "start_node")
				}
				if end == compose.END {
					endID = m.nodeID(prefix, "end_node")
				}
				if style {
					sb.WriteString(fmt.Sprintf("%s%s -. data-only .-> %s\n", indent, startID, endID))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s -.-> %s\n", indent, startID, endID))
				}
			}
		}
	}

	// Render Branches
	branchStarts := make([]string, 0, len(info.Branches))
	for k := range info.Branches {
		branchStarts = append(branchStarts, k)
	}
	sort.Strings(branchStarts)

	for _, start := range branchStarts {
		branches := info.Branches[start]
		for i, branch := range branches {
			// Branch decision node (diamond)
			// We need a unique ID for the decision point if there are multiple branches from the same node?
			// Actually, `info.Branches` maps startNode -> []GraphBranch.
			// Usually a node has one set of branches.
			// Let's represent the branch condition as a diamond.

			// If there are multiple branches, they might be parallel or sequential conditions.
			// Eino `AddBranch` adds a branch.

			// For visualization, maybe we just draw arrows from startNode to endNodes with a label?
			// Or introduce a "decision" node?

			// Decision node visualization: startNode -> decision{branch} -> endNodes

			decisionID := fmt.Sprintf("%s_branch_%d", m.nodeID(prefix, start), i)
			sb.WriteString(fmt.Sprintf("%s%s{\"%s\"}\n", indent, decisionID, "branch"))
			startID := m.nodeID(prefix, start)
			if start == compose.START {
				startID = m.nodeID(prefix, "start_node")
			}
			if style {
				sb.WriteString(fmt.Sprintf("%s%s ==> %s\n", indent, startID, decisionID))
			} else {
				sb.WriteString(fmt.Sprintf("%s%s --> %s\n", indent, startID, decisionID))
			}

			// Sort end nodes
			endNodesMap := branch.GetEndNode()
			endNodes := make([]string, 0, len(endNodesMap))
			for k := range endNodesMap {
				endNodes = append(endNodes, k)
			}
			sort.Strings(endNodes)

			for _, end := range endNodes {
				endID := m.nodeID(prefix, end)
				if end == compose.END {
					endID = m.nodeID(prefix, "end_node")
				}
				if style {
					sb.WriteString(fmt.Sprintf("%s%s ==> %s\n", indent, decisionID, endID))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s --> %s\n", indent, decisionID, endID))
				}
			}
		}
	}
}

// nodeID sanitizes a node key to be a valid Mermaid identifier, and adds a caller-provided prefix
// to ensure uniqueness when rendering nested graphs.
func (m *MermaidGenerator) nodeID(prefix, key string) string {
	// Sanitize key for Mermaid ID
	safeKey := strings.ReplaceAll(key, " ", "_")
	safeKey = strings.ReplaceAll(safeKey, "-", "_")
	return prefix + safeKey
}

func (m *MermaidGenerator) renderImage(input, output string) {
	if _, err := exec.LookPath("mmdc"); err != nil {
		// fallback to chromedp rendering
		data, rErr := os.ReadFile(input)
		if rErr != nil {
			return
		}
		_ = renderWithChromedp(string(data), output)
		return
	}
	cmd := exec.Command("mmdc", "-i", input, "-o", output)
	_ = cmd.Run()
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func renderWithChromedp(mermaidCode, output string) error {
	html := buildMermaidHTML(mermaidCode)
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var err error
	ext := strings.ToLower(filepath.Ext(output))
	switch ext {
	case ".png":
		var buf []byte
		err = chromedp.Run(ctx,
			chromedp.Navigate("data:text/html,"+urlEncode(html)),
			chromedp.WaitVisible(`#container svg`, chromedp.ByQuery),
			chromedp.Screenshot(`#container svg`, &buf, chromedp.NodeVisible, chromedp.ByQuery),
		)
		if err == nil {
			err = os.WriteFile(output, buf, 0644)
		}
	case ".svg":
		var svg string
		err = chromedp.Run(ctx,
			chromedp.Navigate("data:text/html,"+urlEncode(html)),
			chromedp.WaitVisible(`#container svg`, chromedp.ByQuery),
			chromedp.OuterHTML(`#container svg`, &svg, chromedp.ByQuery),
		)
		if err == nil {
			err = os.WriteFile(output, []byte(svg), 0644)
		}
	default:
		// unsupported extension
		return nil
	}
	return err
}

func buildMermaidHTML(code string) string {
	return `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <script src="https://unpkg.com/mermaid@10/dist/mermaid.min.js"></script>
  <style>body{margin:0;padding:20px;background:#fff}</style>
  <script>mermaid.initialize({startOnLoad:true});</script>
</head>
<body>
  <div id="container" class="mermaid">` + code + `</div>
</body>
</html>`
}

func urlEncode(s string) string { // minimal percent-encoding for data URL
	r := strings.NewReplacer("%", "%25", "#", "%23", "\n", "%0A", "\r", "%0D")
	return r.Replace(s)
}
