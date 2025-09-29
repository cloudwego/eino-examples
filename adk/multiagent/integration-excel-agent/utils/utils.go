package utils

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"golang.org/x/sync/errgroup"

	"github.com/cloudwego/eino/adk"
)

type panicErr struct {
	info  any
	stack []byte
}

func (p *panicErr) Error() string {
	return fmt.Sprintf("panic error: %v, \nstack: %s", p.info, string(p.stack))
}

// NewPanicErr creates a new panic error.
// panicErr is a wrapper of panic info and stack trace.
// it implements the error interface, can print error message of info and stack trace.
func NewPanicErr(info any, stack []byte) error {
	return &panicErr{
		info:  info,
		stack: stack,
	}
}

func PtrOf[T any](v T) *T {
	return &v
}

func FormatInput(input []adk.Message) string {
	var sb strings.Builder
	for _, msg := range input {
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

func GenErrorIter(err error) *adk.AsyncIterator[*adk.AgentEvent] {
	iterator, generator := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	generator.Send(&adk.AgentEvent{Err: err})
	generator.Close()
	return iterator
}

type TaskGroup interface {
	Go(f func() error)
	Wait() error
}

type taskGroup struct {
	errGroup    *errgroup.Group
	ctx         context.Context
	execAllTask atomic.Bool
}

// NewTaskGroup if one task return error, the rest task will stop
func NewTaskGroup(ctx context.Context, concurrentCount int) TaskGroup {
	t := &taskGroup{}
	t.errGroup, t.ctx = errgroup.WithContext(ctx)
	t.errGroup.SetLimit(concurrentCount)
	t.execAllTask.Store(false)

	return t
}

func (t *taskGroup) Go(f func() error) {
	t.errGroup.Go(func() error {
		defer func() {
			if err := recover(); err != nil {
				err_ := NewPanicErr(err, debug.Stack())
				fmt.Println(fmt.Errorf("[TaskGroup] exec panic recover:%+v", err_))
			}
		}()

		if !t.execAllTask.Load() {
			select {
			case <-t.ctx.Done():
				return t.ctx.Err()
			default:
			}
		}

		return f()
	})
}

func (t *taskGroup) Wait() error {
	return t.errGroup.Wait()
}

func ToJSONString(v interface{}) string {
	str, _ := sonic.MarshalString(v)
	return str
}

func GetCurrentTime() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 出现错误时 fallback 到本地时间
		return time.Now().Format("2006-01-02 15:04:05 MST")
	}
	return time.Now().In(loc).Format("2006-01-02 15:04:05 MST")
}

func Recovery(ctx context.Context) {
	e := recover()
	if e == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	log.Printf("[utils.Recovery] catch panic!!! err: %+v \nstacktrace:\n%s", e, debug.Stack())
}

func SafeGo(ctx context.Context, fn func()) {
	go func() {
		defer Recovery(ctx)
		fn()
	}()
}

func RepairJSON(input string) string {
	input = strings.TrimPrefix(input, "<|FunctionCallBegin|>")
	input = strings.TrimSuffix(input, "<|FunctionCallEnd|>")
	input = strings.TrimPrefix(input, "<think>")
	s := fixBrackets(input)
	return s
}

func fixBrackets(s string) string {
	var stack []rune
	var result []rune

	for _, ch := range s {
		if ch == '{' || ch == '[' {
			stack = append(stack, ch)
			result = append(result, ch)
		} else if ch == '}' || ch == ']' {
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				if (ch == '}' && top == '{') || (ch == ']' && top == '[') {
					stack = stack[:len(stack)-1] // 匹配成功，出栈
					result = append(result, ch)
				} else {
					continue
				}
			} else {
				continue
			}
		} else {
			result = append(result, ch)
		}
	}

	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			result = append(result, '}')
		} else if stack[i] == '[' {
			result = append(result, ']')
		}
	}

	return string(result)
}
