# 当前功能清单

本文档用于汇总当前后端已实现能力、部署要点与待补齐项。

## 1. 启动与基础能力

- 启动入口：`main.go`
- 依赖初始化：Redis 必需、MySQL 可选
- MySQL 可用时自动执行表迁移（AutoMigrate）
- Gin 模式、Socket CORS、Socket 限流、Redis/MySQL 都支持环境变量配置

## 2. HTTP 接口

- `GET /health`
  - Redis 健康标记与基本存活检查
- `GET /mysql/test-read`
  - MySQL 连通性测试读取
- `GET /users/:id`
  - 用户按 ID 查询
- `GET /conversations/:id/messages`
  - 会话消息分页查询（支持 `beforeId`、`offset`、`limit`）

## 3. Socket 事件

- `bind_user`
  - 参数：`{ userId: string }`
  - 建立 userId 与 socketId 绑定
- `send_to_user`
  - 参数：`{ toUserId: string, message: string }`
  - 单聊消息发送
- `send_to_group`
  - 参数：`{ conversationId: number, message: string, clientMsgId?: string }`
  - 群聊消息发送
- `private_message`
  - 私聊下行事件
- `group_message`
  - 群聊下行事件

## 4. 多实例能力

- 单聊/群聊均支持跨实例转发
- 转发机制：Redis Pub/Sub
- 在线路由：本机连接优先，找不到时走跨实例发布
- 多实例前提：
  - 所有实例连接同一个 Redis
  - 所有实例连接同一个 MySQL
  - `REDIS_KEY_PREFIX` 必须一致

## 5. 生产基线增强

- 每连接发送限流（令牌桶）
  - 覆盖 `send_to_user` 与 `send_to_group`
  - 可配置：`SOCKET_SEND_RATE`、`SOCKET_SEND_BURST`
- 群聊幂等去重
  - 使用 `clientMsgId` + `conversationId` 防止重试重复落库/重复广播
  - 回执字段：`send_to_group_ok.deduplicated`
- 大群分页 fan-out
  - 群成员按分页拉取，避免固定上限造成漏发

## 6. 数据模型与仓储

已落地模型：

- `users`
- `conversations`
- `conversation_members`
- `messages`
- `friendships`
- `friend_requests`

已落地 repository：

- User
- Conversation
- ConversationMember
- Message
- Friendship
- FriendRequest

## 7. 已提供文档与脚本

- 前端接入：`docs/frontend-socket-integration.md`
- 压测脚本：`scripts/loadtest.js`（当前主要覆盖 HTTP 拉消息链路）

## 8. 当前未完成项（建议优先级）

- P1：Socket 鉴权（JWT / token 校验）
- P1：离线消息投递与补偿队列
- P1：消息状态流转（sent/delivered/read）与 ACK 重试
- P2：分布式限流（跨实例统一）
- P2：观测与告警（QPS、失败率、P99、消费者堆积）

## 9. 运行排查提示

- 启动失败先看端口占用：`HTTP_PORT`
- 若怀疑未落表，先确认：
  - `MYSQL_DSN` 指向预期数据库
  - 服务已成功启动并执行迁移
  - GORM 默认复数表名（如 `users`、`messages`）
