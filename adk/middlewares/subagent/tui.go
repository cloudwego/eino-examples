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
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/subagent"
	"github.com/cloudwego/eino/schema"
)

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	toolCallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))

	toolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

	answerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	statusRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	statusCompleted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)

	statusFailed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusCanceled = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	panelHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15"))

	tagToolCall = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("75")).
			Padding(0, 1).
			Render("TOOL CALL")

	tagToolResult = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("114")).
			Padding(0, 1).
			Render("TOOL RESULT")

	tagAnswer = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("229")).
			Padding(0, 1).
			Render("ANSWER")

	tagError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("196")).
			Padding(0, 1).
			Render("ERROR")
)

// --- Tea Messages ---

type mainEventMsg struct {
	entry logEntry
}

type subAgentNotificationMsg struct {
	task *subagent.Task
}

type subAgentEventMsg struct {
	taskID string
	entry  logEntry
}

type queryDoneMsg struct {
	err error
}

// subAgentsDoneMsg signals that background subagents have completed and a new turn should start.
type subAgentsDoneMsg struct {
	summary string // formatted summary of subagent results to feed back
}

// turnCompleteMsg signals that no more subagent results are pending and the user can input again.
type turnCompleteMsg struct{}

type errMsg struct {
	err error
}

// --- Log Entry ---

type logEntry struct {
	Type      string // "toolcall", "toolresult", "answer", "info", "error"
	Agent     string
	Content   string
	Timestamp time.Time
}

// renderFull renders a log entry with full content (no truncation), returning multiple lines.
func (e logEntry) renderFull(width int) []string {
	ts := labelStyle.Render(e.Timestamp.Format("15:04:05"))
	agentTag := ""
	if e.Agent != "" {
		agentTag = " " + labelStyle.Render("<"+e.Agent+">")
	}

	headerPrefix := ts + agentTag + " "

	switch e.Type {
	case "toolcall":
		header := headerPrefix + tagToolCall
		body := toolCallStyle.Render(e.Content)
		return renderBlock(header, body, width)
	case "toolresult":
		header := headerPrefix + tagToolResult
		body := toolResultStyle.Render(e.Content)
		return renderBlock(header, body, width)
	case "answer":
		header := headerPrefix + tagAnswer
		body := answerStyle.Render(e.Content)
		return renderBlock(header, body, width)
	case "error":
		header := headerPrefix + tagError
		body := errorStyle.Render(e.Content)
		return renderBlock(header, body, width)
	case "info":
		return []string{headerPrefix + infoStyle.Render(e.Content)}
	default:
		return []string{headerPrefix + e.Content}
	}
}

// renderBlock outputs a header line followed by body content wrapped to width.
func renderBlock(header, body string, width int) []string {
	lines := []string{header}
	// Indent body by 2 spaces, wrap each line
	bodyWidth := width - 2
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	for _, rawLine := range strings.Split(body, "\n") {
		wrapped := softWrap(rawLine, bodyWidth)
		for _, w := range wrapped {
			lines = append(lines, "  "+w)
		}
	}
	return lines
}

// --- Task Info ---

type taskInfo struct {
	ID          string
	Description string
	Status      subagent.Status
	Result      string
}

func (t taskInfo) StatusStyle() string {
	switch t.Status {
	case subagent.StatusRunning:
		return statusRunning.Render(string(t.Status))
	case subagent.StatusCompleted:
		return statusCompleted.Render(string(t.Status))
	case subagent.StatusFailed:
		return statusFailed.Render(string(t.Status))
	case subagent.StatusCanceled:
		return statusCanceled.Render(string(t.Status))
	default:
		return string(t.Status)
	}
}

// --- TUI Model ---

type tuiModel struct {
	// Main agent panel
	mainLogs   []logEntry
	mainScroll int // offset from bottom (0 = auto-scroll to bottom)

	// SubAgent panel
	subAgentTasks  map[string]*taskInfo
	subAgentLogs   []logEntry
	subAgentScroll int // offset from bottom

	// Input
	textInput textinput.Model
	inputMode bool
	querying  bool

	// Layout
	width  int
	height int
}

func newTUIModel() tuiModel {
	ti := textinput.New()
	ti.Placeholder = "Enter your query..."
	ti.Focus()
	ti.CharLimit = 2048
	ti.Width = 120

	return tuiModel{
		mainLogs:      make([]logEntry, 0),
		subAgentTasks: make(map[string]*taskInfo),
		subAgentLogs:  make([]logEntry, 0),
		textInput:     ti,
		inputMode:     true,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.inputMode && !m.querying {
				input := m.textInput.Value()
				if input == "" {
					return m, nil
				}
				m.textInput.SetValue("")
				m.querying = true
				m.inputMode = false
				m.mainLogs = append(m.mainLogs, logEntry{
					Type:      "info",
					Content:   "You: " + input,
					Timestamp: time.Now(),
				})
				m.mainScroll = 0
				return m, func() tea.Msg {
					return userQueryMsg{query: input}
				}
			}
		case "up":
			m.mainScroll++
		case "down":
			if m.mainScroll > 0 {
				m.mainScroll--
			}
		case "shift+up":
			m.subAgentScroll++
		case "shift+down":
			if m.subAgentScroll > 0 {
				m.subAgentScroll--
			}
		case "tab":
			// Reset scroll to bottom for both panels
			m.mainScroll = 0
			m.subAgentScroll = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 6

	case userQueryMsg:
		return m, nil

	case mainEventMsg:
		m.mainLogs = append(m.mainLogs, msg.entry)
		m.mainScroll = 0 // auto-scroll to bottom on new content

	case subAgentNotificationMsg:
		t := msg.task
		m.subAgentTasks[t.ID] = &taskInfo{
			ID:          t.ID,
			Description: t.Description,
			Status:      t.Status,
			Result:      t.Result,
		}
		if t.Error != "" {
			m.subAgentTasks[t.ID].Result = "ERROR: " + t.Error
		}

	case subAgentEventMsg:
		m.subAgentLogs = append(m.subAgentLogs, msg.entry)
		m.subAgentScroll = 0 // auto-scroll

	case queryDoneMsg:
		if msg.err != nil {
			m.mainLogs = append(m.mainLogs, logEntry{
				Type:      "error",
				Content:   msg.err.Error(),
				Timestamp: time.Now(),
			})
		}
		m.mainLogs = append(m.mainLogs, logEntry{
			Type:      "info",
			Content:   "--- Turn complete ---",
			Timestamp: time.Now(),
		})
		m.mainScroll = 0
		// Don't return to input mode here; the appModel decides whether
		// to wait for subagents or return to input.

	case subAgentsDoneMsg:
		m.mainLogs = append(m.mainLogs, logEntry{
			Type:      "info",
			Content:   "SubAgent results received, starting next turn...",
			Timestamp: time.Now(),
		})
		m.mainScroll = 0

	case turnCompleteMsg:
		// All turns done (no more pending subagent results), return to input
		m.querying = false
		m.inputMode = true
		m.textInput.Focus()
		m.mainLogs = append(m.mainLogs, logEntry{
			Type:      "info",
			Content:   "--- Ready for input ---",
			Timestamp: time.Now(),
		})
		m.mainScroll = 0
		return m, textinput.Blink

	case errMsg:
		m.mainLogs = append(m.mainLogs, logEntry{
			Type:      "error",
			Content:   msg.err.Error(),
			Timestamp: time.Now(),
		})
	}

	if m.inputMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

type userQueryMsg struct {
	query string
}

func (m tuiModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	w := m.width
	// Layout: title(1) + subagent panel (top) + main panel (bottom) + input(1)
	// Each panel has 2 lines border overhead
	availHeight := m.height - 3 // title + input + gap
	if availHeight < 6 {
		availHeight = 6
	}
	// SubAgent panel (top) gets 40%, Main panel (bottom) gets 60%
	subPanelH := availHeight * 40 / 100
	mainPanelH := availHeight - subPanelH
	if mainPanelH < 4 {
		mainPanelH = 4
	}
	if subPanelH < 3 {
		subPanelH = 3
	}

	// Reserve columns for border(2) + scrollbar(1)
	panelContentW := w - 3
	if panelContentW < 20 {
		panelContentW = 20
	}

	// Title
	title := titleStyle.Render(" SubAgent Example - Eino ADK ") +
		"  " + labelStyle.Render("[shift+up/down] scroll subagent  [up/down] scroll main  [tab] reset  [ctrl+c] quit")

	// SubAgent panel (top)
	subRendered := m.renderSubAgentLines(panelContentW)
	subViewH := subPanelH - 4 // border + header + sep
	if subViewH < 1 {
		subViewH = 1
	}
	subVisible := scrollView(subRendered, subViewH, m.subAgentScroll)
	subScrollbar := renderScrollbar(len(subRendered), subViewH, m.subAgentScroll)
	subContent := m.buildSubAgentHeader(panelContentW) + "\n" + joinWithScrollbar(subVisible, subScrollbar, subViewH)
	subPanel := panelBorderStyle.
		Width(panelContentW + 1). // +1 for scrollbar column
		Height(subPanelH).
		Render(subContent)

	// Main agent panel (bottom)
	mainRendered := m.renderMainLines(panelContentW)
	mainViewH := mainPanelH - 4 // border + header + sep
	if mainViewH < 1 {
		mainViewH = 1
	}
	mainVisible := scrollView(mainRendered, mainViewH, m.mainScroll)
	mainScrollbar := renderScrollbar(len(mainRendered), mainViewH, m.mainScroll)
	mainContent := m.buildMainHeader(panelContentW) + "\n" + joinWithScrollbar(mainVisible, mainScrollbar, mainViewH)
	mainPanel := panelBorderStyle.
		Width(panelContentW + 1).
		Height(mainPanelH).
		Render(mainContent)

	// Input bar
	var inputBar string
	if m.inputMode {
		inputBar = inputPromptStyle.Render(">> ") + m.textInput.View()
	} else if m.querying {
		inputBar = infoStyle.Render("  Agent is working...")
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, subPanel, mainPanel, inputBar)
}

// buildMainHeader returns the header + separator for the main panel.
func (m tuiModel) buildMainHeader(width int) string {
	header := panelHeaderStyle.Render("Main Agent")
	if m.querying {
		header += "  " + statusRunning.Render("working...")
	}
	sep := labelStyle.Render(strings.Repeat("─", width))
	return header + "\n" + sep
}

// renderMainLines returns all rendered content lines for the main panel (no header).
func (m tuiModel) renderMainLines(width int) []string {
	var rendered []string
	for _, entry := range m.mainLogs {
		rendered = append(rendered, entry.renderFull(width)...)
	}
	return rendered
}

// buildSubAgentHeader returns the header + separator for the subagent panel.
func (m tuiModel) buildSubAgentHeader(width int) string {
	header := panelHeaderStyle.Render("Background SubAgents")
	taskCount := len(m.subAgentTasks)
	runningCount := 0
	for _, t := range m.subAgentTasks {
		if t.Status == subagent.StatusRunning {
			runningCount++
		}
	}
	if taskCount > 0 {
		header += "  " + labelStyle.Render(fmt.Sprintf("(%d tasks, %d running)", taskCount, runningCount))
	}
	sep := labelStyle.Render(strings.Repeat("─", width))
	return header + "\n" + sep
}

// renderSubAgentLines returns all rendered content lines for the subagent panel (no header).
func (m tuiModel) renderSubAgentLines(width int) []string {
	var rendered []string

	// Task status section
	if len(m.subAgentTasks) > 0 {
		for _, t := range m.subAgentTasks {
			taskLine := fmt.Sprintf("[%s] %s  %s", t.ID, t.StatusStyle(), t.Description)
			rendered = append(rendered, taskLine)
			if t.Status == subagent.StatusCompleted && t.Result != "" {
				for _, rl := range softWrap(t.Result, width-4) {
					rendered = append(rendered, "    "+infoStyle.Render(rl))
				}
			}
			if t.Status == subagent.StatusFailed && t.Result != "" {
				for _, rl := range softWrap(t.Result, width-4) {
					rendered = append(rendered, "    "+errorStyle.Render(rl))
				}
			}
		}
		rendered = append(rendered, labelStyle.Render(strings.Repeat("─", width)))
	}

	// Event log
	if len(m.subAgentLogs) > 0 {
		rendered = append(rendered, panelHeaderStyle.Render("Event Log:"))
		for _, entry := range m.subAgentLogs {
			rendered = append(rendered, entry.renderFull(width)...)
		}
	} else if len(m.subAgentTasks) == 0 {
		rendered = append(rendered, infoStyle.Render("No background tasks yet. SubAgents will appear here when spawned."))
	}

	return rendered
}

// renderScrollbar generates a column of scrollbar characters for the given panel.
// It returns a slice of single-char strings, one per visible row.
func renderScrollbar(totalLines, viewH, scrollFromBottom int) []string {
	bar := make([]string, viewH)
	if totalLines <= viewH {
		// No scrollbar needed, fill with spaces
		for i := range bar {
			bar[i] = " "
		}
		return bar
	}

	// Calculate thumb position and size
	thumbSize := viewH * viewH / totalLines
	if thumbSize < 1 {
		thumbSize = 1
	}

	// scrollFromBottom=0 means we see the bottom; scrollFromBottom=max means we see the top
	maxScroll := totalLines - viewH
	scrollFromTop := maxScroll - scrollFromBottom
	if scrollFromTop < 0 {
		scrollFromTop = 0
	}
	if scrollFromTop > maxScroll {
		scrollFromTop = maxScroll
	}

	thumbPos := 0
	if maxScroll > 0 {
		thumbPos = scrollFromTop * (viewH - thumbSize) / maxScroll
	}

	scrollTrack := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	scrollThumb := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))

	for i := 0; i < viewH; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			bar[i] = scrollThumb.Render("┃")
		} else {
			bar[i] = scrollTrack.Render("│")
		}
	}
	return bar
}

// joinWithScrollbar combines content lines with a scrollbar column on the right.
// Pads content to exactly viewH lines.
func joinWithScrollbar(lines []string, scrollbar []string, viewH int) string {
	// Pad lines to viewH
	for len(lines) < viewH {
		lines = append(lines, "")
	}
	var result []string
	for i := 0; i < viewH; i++ {
		sb := " "
		if i < len(scrollbar) {
			sb = scrollbar[i]
		}
		result = append(result, lines[i]+sb)
	}
	return strings.Join(result, "\n")
}

// scrollView returns a window of `viewH` lines from `all`, scrolled from the bottom
// by `scrollFromBottom` lines. 0 means show the latest content.
func scrollView(all []string, viewH, scrollFromBottom int) []string {
	total := len(all)
	if total <= viewH {
		return all
	}
	end := total - scrollFromBottom
	if end > total {
		end = total
	}
	if end < viewH {
		end = viewH
	}
	start := end - viewH
	if start < 0 {
		start = 0
		end = viewH
	}
	if end > total {
		end = total
	}
	return all[start:end]
}

// softWrap wraps text at `width` display columns, preserving existing newlines.
// Uses runewidth to correctly handle CJK and other wide characters.
func softWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var result []string
	for _, line := range strings.Split(text, "\n") {
		if runewidth.StringWidth(line) <= width {
			result = append(result, line)
			continue
		}
		for len(line) > 0 {
			var cut int
			var w int
			for i, r := range line {
				rw := runewidth.RuneWidth(r)
				if w+rw > width {
					cut = i
					break
				}
				w += rw
				cut = i + utf8.RuneLen(r)
			}
			if cut == 0 {
				// Single character wider than width, take at least one rune
				_, size := utf8.DecodeRuneInString(line)
				cut = size
			}
			result = append(result, line[:cut])
			line = line[cut:]
		}
	}
	return result
}

// --- Event processing ---

// processAgentEvent converts an AgentEvent into TUI messages for the main panel.
// Returns the resolved message (if any) for history accumulation.
func processAgentEvent(event *adk.AgentEvent, p *tea.Program) *schema.Message {
	agentName := event.AgentName

	if event.Output != nil && event.Output.MessageOutput != nil {
		m, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			p.Send(errMsg{err: err})
			return nil
		}
		if m == nil {
			return nil
		}
		if len(m.Content) > 0 {
			entryType := "answer"
			if m.Role == schema.Tool {
				entryType = "toolresult"
			}
			p.Send(mainEventMsg{entry: logEntry{
				Type:      entryType,
				Agent:     agentName,
				Content:   m.Content,
				Timestamp: time.Now(),
			}})
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				p.Send(mainEventMsg{entry: logEntry{
					Type:      "toolcall",
					Agent:     agentName,
					Content:   fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments),
					Timestamp: time.Now(),
				}})
			}
		}
		return m
	}

	if event.Action != nil {
		if event.Action.TransferToAgent != nil {
			p.Send(mainEventMsg{entry: logEntry{
				Type:      "info",
				Agent:     agentName,
				Content:   fmt.Sprintf("Transfer to agent: %s", event.Action.TransferToAgent.DestAgentName),
				Timestamp: time.Now(),
			}})
		}
		if event.Action.Exit {
			p.Send(mainEventMsg{entry: logEntry{
				Type:      "info",
				Agent:     agentName,
				Content:   "Agent exit",
				Timestamp: time.Now(),
			}})
		}
	}
	return nil
}

// processSubAgentEvent converts a subagent's AgentEvent into TUI messages for the subagent panel.
func processSubAgentEvent(event *adk.AgentEvent, taskID string, p *tea.Program) {
	agentName := event.AgentName

	if event.Output != nil && event.Output.MessageOutput != nil {
		m, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			p.Send(subAgentEventMsg{taskID: taskID, entry: logEntry{
				Type:      "error",
				Agent:     agentName,
				Content:   err.Error(),
				Timestamp: time.Now(),
			}})
			return
		}
		if m == nil {
			return
		}
		if len(m.Content) > 0 {
			entryType := "answer"
			if m.Role == schema.Tool {
				entryType = "toolresult"
			}
			p.Send(subAgentEventMsg{taskID: taskID, entry: logEntry{
				Type:      entryType,
				Agent:     agentName,
				Content:   m.Content,
				Timestamp: time.Now(),
			}})
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				p.Send(subAgentEventMsg{taskID: taskID, entry: logEntry{
					Type:      "toolcall",
					Agent:     agentName,
					Content:   fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments),
					Timestamp: time.Now(),
				}})
			}
		}
	}
}
