## 说明

### 文档地址
[eino 实践教程](https://www.cloudwego.io/zh/docs/eino/overview/bytedance_eino_practice/)

### docker 启动 redis 作为向量数据库

```bash
# Notice: 如果本地已经安装了 redis，则需要先停止本地的 redis 容器，再执行上述命令，否则端口冲突会导致启动失败
docker-compose up -d
# 可以在 http://127.0.0.1:8001 看到 redis 的 web 界面
# redis 监听在 127.0.0.1:6379, 使用 redis-cli ping 可测试
```

### 环境变量

#### 大模型

所需的大模型和 API Key.
豆包大模型地址: https://console.volcengine.com/ark/region:ark+cn-beijing/model
> ChatModel 推荐: [Doubao-pro-4k (functioncall)](https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-4k)
> EmbeddingModel 推荐: [Doubao-embedding-large](https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-embedding-large)
> 进入页面后点击 `推理` 按钮，即可创建按量计费的模型接入点，对应的 `ep-xxx` 就是所需的 model 名称

```bash
export ARK_API_KEY=xxx
export ARK_CHAT_MODEL=xxx
export ARK_EMBEDDING_MODEL=xxx
```

#### 可观测

可观测能力的自动集成，仅需配置下方环境变量，即可访问APMPlus查看观测数据: https://console.volcengine.com/apmplus-server
```bash
export APMPLUS_APP_KEY=xxx
export APMPLUS_REGION=xxx # 选填
```

### 启动 eino agent server

#### 命令行方式

```bash
# 为了使用 data 目录，需要在 eino_assistant 目录下执行指令
go run cmd/einoagent/main.go
```

#### 容器方式
```bash
docker build --platform=linux/amd64 . -t eino-assistant:latest
docker run -p 8080:8080 -e ARK_API_KEY=xxx -e ARK_CHAT_MODEL=xxx -e ARK_EMBEDDING_MODEL=xxx -e APMPLUS_APP_KEY=xxx eino-assistant:latest
```

### 访问

访问 http://127.0.0.1:8080/ 即可看到效果

### 命令行运行 index (可选)

```bash
# 因示例的Markdown文件存放在 cmd/knowledgeindexing/eino-docs 目录，代码中指定了相对路径 ./eino-docs，所以需在 cmd/knowledgeindexing 运行指令
cd cmd/knowledgeindexing
go run main.go
```
