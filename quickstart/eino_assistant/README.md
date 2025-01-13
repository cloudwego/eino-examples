## 说明

### docker 启动 redis 作为向量数据库

```bash
docker-compose up -d
# 可以在 http://127.0.0.1:8001 看到 redis 的 web 界面
# redis 监听在 127.0.0.1:6379, 使用 redis-cli ping 可测试
```

### 环境变量

所需的大模型和 API Key.
豆包大模型地址: https://console.volcengine.com/ark/region:ark+cn-beijing/model
> ChatModel 推荐: [Doubao-pro-32k](https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-32k)
> EmbeddingModel 推荐: [Doubao-embedding-large](https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-embedding-large)
> 进入页面后点击 `推理` 按钮，即可创建按量计费的模型接入点，对应的 `ep-xxx` 就是所需的 model 名称

```bash
export ARK_API_KEY=xxx
export ARK_CHAT_MODEL=xxx
export ARK_EMBEDDING_MODEL=xxx
```

### 启动 server

```bash
go build -o einoagent cmd/server/main.go && ./einoagent
```

### 访问

访问 http://127.0.0.1:8080/ 即可看到效果

### 命令行运行 index (可选)

```bash
go run cmd/index/main.go
```

### 命令行运行 agent (可选)

```bash
go run cmd/agent/main.go
```
