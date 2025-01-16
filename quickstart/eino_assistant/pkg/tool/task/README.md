# Task Tool

一个简单的 Task 管理工具，支持 Web 界面和 API 接口。

## 功能特点

- 支持添加、更新、删除和列表查询
- 支持按标题和内容搜索
- 支持按完成状态筛选
- 支持软删除
- 支持按创建时间排序
- 数据持久化到本地文件
- 美观的 Web 界面
- 实时自动更新

## 启动服务

```bash
go run cmd/web/main.go
```

服务默认在 8080 端口启动，可以通过环境变量 `PORT` 修改：

```bash
PORT=3000 go run cmd/web/main.go
```

## API 使用示例

### 添加 Task

```bash
curl -X POST http://127.0.0.1:8080/task/api \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add",
    "task": {
      "title": "完成作业",
      "content": "完成数学作业",
      "deadline": "2024-01-15T18:00:00Z"
    }
  }'
```

### 更新 Task

```bash
curl -X POST http://127.0.0.1:8080/task/api \
  -H "Content-Type: application/json" \
  -d '{
    "action": "update",
    "task": {
      "id": "task-id",
      "completed": true
    }
  }'
```

### 删除 Task

```bash
curl -X POST http://127.0.0.1:8080/task/api \
  -H "Content-Type: application/json" \
  -d '{
    "action": "delete",
    "task": {
      "id": "task-id"
    }
  }'
```

### 列出所有 Task

```bash
curl -X POST http://127.0.0.1:8080/task/api \
  -H "Content-Type: application/json" \
  -d '{
    "action": "list",
    "list": {}
  }'
```

### 搜索和筛选 Task

```bash
curl -X POST http://127.0.0.1:8080/task/api \
  -H "Content-Type: application/json" \
  -d '{
    "action": "list",
    "list": {
      "query": "作业",
      "is_done": false,
      "limit": 10
    }
  }'
```

## API 响应格式

所有 API 响应都遵循以下格式：

```json
{
  "status": "success",
  "task_list": [
    {
      "id": "uuid",
      "title": "标题",
      "content": "内容",
      "completed": false,
      "deadline": "2024-01-15T18:00:00Z",
      "created_at": "2024-01-10T10:00:00Z"
    }
  ],
  "error": ""
}
```

- `status`: 可能的值为 "success" 或 "error"
- `task_list`: Task 项列表，某些操作可能为空
- `error`: 错误信息，成功时为空

## 数据存储

Task 数据以 JSON Lines 格式存储在 `.task/tasks.jsonl` 文件中。每行一个 Task 项，支持实时读写。 