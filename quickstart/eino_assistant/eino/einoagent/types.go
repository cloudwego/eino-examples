package einoagent

import "github.com/cloudwego/eino/schema"

type UserMessage struct {
	ID      string            `json:"id"`
	Query   string            `json:"query"`
	History []*schema.Message `json:"history"`
}
