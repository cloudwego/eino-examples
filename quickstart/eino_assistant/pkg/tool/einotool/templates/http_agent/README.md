# http_agent

## 简介

http_agent 是一个基于 eino 的 http 服务构建的一个简单的 llm 应用。

## 使用

### 启动 http server

```bash
go run main.go -model=ep-xxxx -apikey=xxx
```

### 使用 curl 访问 http server

```bash
curl 'http://127.0.0.1:8888/chat?id=123&msg=hello'
```
> 注意，由于采用了 sse 的格式，结果中会有 `data:` 前缀

### 使用 client

client 是一个简单的交互式客户端，可以与 http server 进行交互，并打印结果。

```bash
go run client/main.go
```
