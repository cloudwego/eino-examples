package todo

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
)

type Action string

const (
	ActionAdd    Action = "add"
	ActionGet    Action = "get"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
)

type Todo struct {
	ID        string `json:"id" jsonschema:"description:id of the todo"`
	Title     string `json:"title" jsonschema:"description:title of the todo"`
	Content   string `json:"content" jsonschema:"description:content of the todo"`
	Completed bool   `json:"completed" jsonschema:"description:completed status of the todo"`
	Deadline  string `json:"deadline" jsonschema:"description:deadline of the todo"`
	IsDeleted bool   `json:"is_deleted" jsonschema:"-"`

	CreatedAt string `json:"created_at" jsonschema:"description:created time of the todo"`
}

type TodoRequest struct {
	Action Action      `json:"action" jsonschema:"description:action to perform, enum:add,update,delete,list"`
	Todo   *Todo       `json:"todo" jsonschema:"description:todo to add, update, or delete"`
	List   *ListParams `json:"list" jsonschema:"description:list parameters"`
}

type ListParams struct {
	Query  string `json:"query" jsonschema:"description:query to search"`
	IsDone *bool  `json:"is_done" jsonschema:"description:filter by completed status"`
	Limit  *int   `json:"limit" jsonschema:"description:limit the number of results"`
}

type TodoResponse struct {
	Status string `json:"status" jsonschema:"description:status of the response"`

	TodoList []*Todo `json:"todo_list" jsonschema:"description:list of todos"`

	Error string `json:"error" jsonschema:"description:error message"`
}

type TodoToolImpl struct {
	config *TodoToolConfig
}

type TodoToolConfig struct {
	Storage *Storage
}

func defaultTodoToolConfig(ctx context.Context) (*TodoToolConfig, error) {
	config := &TodoToolConfig{
		Storage: GetDefaultStorage(),
	}
	return config, nil
}

func NewTodoToolImpl(ctx context.Context, config *TodoToolConfig) (*TodoToolImpl, error) {
	var err error
	if config == nil {
		config, err = defaultTodoToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	t := &TodoToolImpl{config: config}

	return t, nil
}

func NewTodoTool(ctx context.Context, config *TodoToolConfig) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultTodoToolConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage cannot be empty")
	}

	t := &TodoToolImpl{config: config}
	tn, err = t.ToEinoTool()
	if err != nil {
		return nil, err
	}
	return tn, nil
}

func (t *TodoToolImpl) ToEinoTool() (tool.BaseTool, error) {
	return utils.InferTool("todo", "todo tool, you can add, get, update, delete, list todos", t.Invoke)
}

func (t *TodoToolImpl) Invoke(ctx context.Context, req *TodoRequest) (res *TodoResponse, err error) {
	res = &TodoResponse{}

	switch req.Action {
	case ActionAdd:
		if req.Todo == nil {
			res.Status = "error"
			res.Error = "todo is required for add action"
			return res, nil
		}
		if req.Todo.Title == "" {
			res.Status = "error"
			res.Error = "title is required"
			return res, nil
		}
		req.Todo.ID = uuid.New().String()
		if err := t.config.Storage.Add(req.Todo); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to add todo: %v", err)
			return res, nil
		}
		res.TodoList = []*Todo{req.Todo}

	case ActionUpdate:
		if req.Todo == nil {
			res.Status = "error"
			res.Error = "todo is required for update action"
			return res, nil
		}
		if req.Todo.ID == "" {
			res.Status = "error"
			res.Error = "id is required"
			return res, nil
		}
		if err := t.config.Storage.Update(req.Todo); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to update todo: %v", err)
			return res, nil
		}
		res.TodoList = []*Todo{req.Todo}

	case ActionDelete:
		if req.Todo == nil || req.Todo.ID == "" {
			res.Status = "error"
			res.Error = "todo id is required for delete action"
			return res, nil
		}
		if err := t.config.Storage.Delete(req.Todo.ID); err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to delete todo: %v", err)
			return res, nil
		}

	case ActionList:
		if req.List == nil {
			req.List = &ListParams{}
		}
		todos, err := t.config.Storage.List(req.List)
		if err != nil {
			res.Status = "error"
			res.Error = fmt.Sprintf("failed to list todos: %v", err)
			return res, nil
		}
		res.TodoList = todos

	default:
		res.Status = "error"
		res.Error = fmt.Sprintf("unknown action: %s", req.Action)
	}

	res.Status = "success"
	return res, nil
}
