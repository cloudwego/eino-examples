package myagent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func RunMyagentCmd() *cobra.Command {
	var opts commandOptions

	cmd := &cobra.Command{
		Use:   "myagent",
		Short: "运行带 workspace/session 的本地 Agent CLI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runChatCommand(cmd, opts)
		},
	}

	bindSharedFlags(cmd.PersistentFlags(), &opts)
	cmd.AddCommand(newChatCommand(&opts))
	cmd.AddCommand(newRunCommand(&opts))
	cmd.AddCommand(newSessionCommand(&opts))
	cmd.AddCommand(newSkillCommand(&opts))

	return cmd
}

type commandOptions struct {
	Workspace     string
	SessionID     string
	Instruction   string
	Model         string
	BaseURL       string
	Query         string
	MaxIterations int
}

func newChatCommand(shared *commandOptions) *cobra.Command {
	local := *shared
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "进入交互式聊天模式",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runChatCommand(cmd, local)
		},
	}
	bindSharedFlags(cmd.Flags(), &local)
	return cmd
}

func newRunCommand(shared *commandOptions) *cobra.Command {
	local := *shared
	cmd := &cobra.Command{
		Use:   "run [-q query] [query]",
		Short: "执行单轮 agent 请求",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(local.Query)
			if query == "" {
				query = strings.TrimSpace(strings.Join(args, " "))
			}
			if query == "" {
				return errors.New("query 不能为空，请通过 -q 或位置参数传入")
			}
			return runSingleTurn(cmd, local, query)
		},
	}
	bindSharedFlags(cmd.Flags(), &local)
	cmd.Flags().StringVarP(&local.Query, "query", "q", "", "单轮提问内容")
	return cmd
}

func newSessionCommand(shared *commandOptions) *cobra.Command {
	local := *shared
	cmd := &cobra.Command{
		Use:   "session",
		Short: "管理 session",
	}
	bindSessionFlags(cmd.PersistentFlags(), &local)

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出当前 workspace 的全部 session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := newRuntime(cmd.Context(), local)
			if err != nil {
				return err
			}
			metas, err := rt.store.ListSessions()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(metas) == 0 {
				fmt.Fprintln(out, "当前 workspace 还没有 session")
				return nil
			}
			for _, meta := range metas {
				fmt.Fprintf(out, "%s\tmessages=%d\tupdated=%s\tsummary=%s\n",
					meta.SessionID,
					meta.Count,
					meta.UpdatedAt.Format("2006-01-02 15:04:05"),
					trimForDisplay(meta.Summary, 48),
				)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "清空当前 session 的历史与 summary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := newRuntime(cmd.Context(), local)
			if err != nil {
				return err
			}
			if local.SessionID == "" {
				return errors.New("session-id 不能为空")
			}
			if err := rt.store.ClearSession(local.SessionID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已清空 session: %s\n", local.SessionID)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "删除当前 session 的全部持久化文件",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := newRuntime(cmd.Context(), local)
			if err != nil {
				return err
			}
			if local.SessionID == "" {
				return errors.New("session-id 不能为空")
			}
			if err := rt.store.DeleteSession(local.SessionID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已删除 session: %s\n", local.SessionID)
			return nil
		},
	})

	return cmd
}

func bindSharedFlags(flags *pflag.FlagSet, opts *commandOptions) {
	bindSessionFlags(flags, opts)
	flags.StringVarP(&opts.Workspace, "workspace", "w", "", "workspace 目录，默认当前目录下的 myagent_workspace")
	flags.StringVar(&opts.Instruction, "instruction", "", "覆盖默认 system instruction")
	flags.StringVar(&opts.Model, "model", "", "模型名称")
	flags.StringVar(&opts.BaseURL, "base-url", "", "openai base url")
	flags.IntVar(&opts.MaxIterations, "max-iterations", 12, "agent loop 最大迭代次数")
}

func bindSessionFlags(flags *pflag.FlagSet, opts *commandOptions) {
	flags.StringVar(&opts.SessionID, "session-id", "", "session id；为空时自动创建")
}

func runChatCommand(cmd *cobra.Command, opts commandOptions) error {
	rt, err := newRuntime(cmd.Context(), opts)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	printWelcome(out, rt)

	scanner := bufio.NewScanner(cmd.InOrStdin())
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		fmt.Fprint(out, "\nmyagent> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return scanner.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if handled, err := handleSlashCommand(out, rt, line); handled {
			if err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
			if err == errExitChat {
				return nil
			}
			continue
		}

		if err := rt.RunTurn(context.Background(), line, out); err != nil {
			fmt.Fprintf(out, "\nerror: %v\n", err)
		}
	}
}

func runSingleTurn(cmd *cobra.Command, opts commandOptions, query string) error {
	rt, err := newRuntime(cmd.Context(), opts)
	if err != nil {
		return err
	}
	printWelcome(cmd.OutOrStdout(), rt)
	return rt.RunTurn(context.Background(), query, cmd.OutOrStdout())
}

func newSkillCommand(shared *commandOptions) *cobra.Command {
	local := *shared
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "管理 skills",
	}
	cmd.Flags().StringVarP(&local.Workspace, "workspace", "w", "", "workspace 目录")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出所有来源目录中的 skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspacePath := strings.TrimSpace(local.Workspace)
			if workspacePath == "" {
				cwd, err := filepath.Abs(".")
				if err != nil {
					return fmt.Errorf("获取当前目录失败: %w", err)
				}
				workspacePath = filepath.Join(cwd, "myagent_workspace")
			}

			cb := NewContextBuilder(workspacePath)
			entries, err := listAllSkills(cb.skillsLoader)
			if err != nil {
				return err
			}
			return printSkillsList(cmd.OutOrStdout(), entries)
		},
	})

	return cmd
}

var errExitChat = errors.New("exit chat")

func handleSlashCommand(out io.Writer, rt *runtime, line string) (bool, error) {
	switch strings.TrimSpace(line) {
	case "/exit", "exit", "quit", "/quit":
		fmt.Fprintln(out, "bye")
		return true, errExitChat
	case "/help":
		fmt.Fprintln(out, "slash commands: /help /skills /session /history /clear /exit")
		return true, nil
	case "/skills":
		cb := NewContextBuilder(rt.workspace.root)
		entries, err := listAllSkills(cb.skillsLoader)
		if err != nil {
			return true, err
		}
		return true, printSkillsList(out, entries)
	case "/session":
		fmt.Fprintf(out, "session=%s workspace=%s\n", rt.sessionID, rt.workspace.root)
		return true, nil
	case "/history":
		history, err := rt.store.GetHistory(rt.sessionID)
		if err != nil {
			return true, err
		}
		fmt.Fprintf(out, "history messages: %d\n", len(history))
		return true, nil
	case "/clear":
		if err := rt.store.ClearSession(rt.sessionID); err != nil {
			return true, err
		}
		fmt.Fprintf(out, "已清空 session=%s\n", rt.sessionID)
		return true, nil
	default:
		return false, nil
	}
}

func printSkillsList(out io.Writer, entries []skillDirEntry) error {
	totalSkills := 0
	for _, entry := range entries {
		totalSkills += len(entry.Skills)
	}
	if totalSkills == 0 {
		_, err := fmt.Fprintln(out, "未发现任何 skill")
		return err
	}

	for _, entry := range entries {
		if len(entry.Skills) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(out, "\n[%s]\n", entry.Label); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "  path: %s\n", entry.Path); err != nil {
			return err
		}
		for _, sk := range entry.Skills {
			if _, err := fmt.Fprintf(out, "  %-28s %s\n", sk.Name, trimForDisplay(sk.Summary, 80)); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(out, "\ntotal: %d skill(s)\n", totalSkills)
	return err
}
