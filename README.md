# 拿来主义

## 文档

- 前端 Socket 对接: docs/frontend-socket-integration.md
- 当前功能清单: docs/current-capabilities.md

## 新增接口

- Socket 单聊: send_to_user
- Socket 群聊: send_to_group
- HTTP 消息查询: GET /conversations/:id/messages

## 压测脚本

仓库提供了 k6 示例脚本: scripts/loadtest.js

使用方式:

1. 安装 k6
2. 启动服务
3. 运行压测

示例命令:

```
BASE_URL=http://127.0.0.1:3003 CONVERSATION_ID=1 k6 run scripts/loadtest.js
```

当前脚本主要覆盖 HTTP 拉消息链路压测。Socket 并发压测建议后续补充专门脚本（例如 Artillery 或 k6 websocket 场景）。

## 准备工作

```
go version go1.26.2 darwin/arm64

go mod init im_backend

# 清理无用依赖
go mod tidy

# 对齐代码
go fmt ./...
```

## 环境变量

可复制 `.env.example` 为 `.env`，应用启动时会自动读取。

- `HTTP_PORT`: HTTP 服务端口，默认 `3003`
- `GIN_MODE`: Gin 运行模式，默认 `debug`
- `SOCKET_CORS_ORIGIN`: Socket CORS 来源，默认 `*`
- `SOCKET_SEND_RATE`: 每个连接每秒发送消息速率限制（send_to_user/send_to_group），默认 `40`
- `SOCKET_SEND_BURST`: 每个连接突发令牌数，默认 `80`
- `MYSQL_DSN`: MySQL 连接串，默认空（为空时不启用 MySQL）
- `REDIS_ADDR`: Redis 地址，默认 `127.0.0.1:6379`
- `REDIS_USERNAME`: Redis 用户名，启用 ACL 时使用，默认空
- `REDIS_PASSWORD`: Redis 密码，默认空
- `REDIS_DB`: Redis 数据库编号，默认 `0`
- `REDIS_KEY_PREFIX`: Redis key 前缀，默认 `im_backend`
