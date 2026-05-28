# 前端 Socket 对接说明

本文档描述前端如何对接当前 IM 后端的 Socket 能力。

## 连接信息

- 服务地址: http://127.0.0.1:3003
- Socket 路径: /socket.io
- 传输协议: Socket.IO

## 事件协议

### 1) 绑定用户

- 发送事件: bind_user
- 参数要求: 必须是对象，且包含 userId
- 示例参数:
  - { "userId": "1001" }
- 成功回执: bind_user_ok
  - { "userId": "1001" }
- 失败回执: bind_user_error
  - payload should be object: { userId }

注意:

- 当前后端已不再接受字符串形式参数，例如 bind_user, "u1001" 会失败。
- 群聊 send_to_group 会把 userId 解析为数字用户 ID，请使用纯数字字符串（例如 "1001"）。

### 2) 给指定用户发消息

- 发送事件: send_to_user
- 参数要求: 对象，且同时包含 toUserId 和 message
- 示例参数:
  - { "toUserId": "u1002", "message": "hello" }
- 成功回执: send_to_user_ok
  - { "toUserId": "u1002" }
- 失败回执: send_to_user_error
  - missing payload
  - payload should contain toUserId and message
  - target user offline
  - rate limit exceeded

### 3) 接收私聊消息

- 接收事件: private_message
- 事件参数:
  - { "message": "hello" }

### 4) 群聊发送消息

- 发送事件: send_to_group
- 参数要求: 对象，且至少包含 conversationId 和 message
- 示例参数:
  - { "conversationId": 1, "message": "hello group", "clientMsgId": "web-1" }
- 成功回执: send_to_group_ok
  - { "conversationId": 1, "messageId": 1001, "recipientCount": 3, "deduplicated": false }
- 失败回执: send_to_group_error
  - missing payload
  - payload should contain conversationId and message
  - bind_user required
  - sender not in conversation
  - rate limit exceeded

说明:

- send_to_group 会先写入 message 表，再向会话成员分发。
- 若目标成员在其他服务实例，后端会通过 Redis Pub/Sub 跨实例转发。
- 当带上 clientMsgId 重试时，后端会做幂等去重，并在 send_to_group_ok 里返回 deduplicated=true。

### 5) 接收群聊消息

- 接收事件: group_message
- 事件参数:
  - { "conversationId": 1, "fromUserId": "1001", "messageId": 1001, "message": "hello group", "createdAt": "2026-05-28T12:00:00Z" }

### 6) 通用消息

- 发送事件: message
- 参数要求: 字符串
- 示例参数:
  - "test message"
- 回执事件: reply
  - "ok: test message"

### 7) 拉取会话消息（HTTP）

- 方法: GET
- 路径: /conversations/:id/messages
- 查询参数:
  - beforeId: 可选，按消息 ID 向前翻页
  - offset: 可选，默认 0
  - limit: 可选，默认 20
- 示例:
  - /conversations/1/messages?limit=20
- 返回:
  - { "items": [ ...message ] }

## 前端最小示例 (JavaScript)

先安装依赖:

- npm i socket.io-client

示例代码:

```js
import { io } from "socket.io-client";

const socket = io("http://127.0.0.1:3003", {
  path: "/socket.io",
});

socket.on("connect", () => {
  console.log("connected", socket.id);

  // 1) 绑定当前登录用户
  socket.emit("bind_user", { userId: "1001" });

  // 2) 给 1002 发一条私聊
  socket.emit("send_to_user", {
    toUserId: "1002",
    message: "hello from 1001",
  });

  // 3) 发群聊消息
  socket.emit("send_to_group", {
    conversationId: 1,
    message: "hello group",
  });

  // 4) 发通用 message
  socket.emit("message", "ping");
});

socket.on("bind_user_ok", (data) => {
  console.log("bind_user_ok", data);
});

socket.on("bind_user_error", (err) => {
  console.error("bind_user_error", err);
});

socket.on("send_to_user_ok", (data) => {
  console.log("send_to_user_ok", data);
});

socket.on("send_to_user_error", (err) => {
  console.error("send_to_user_error", err);
});

socket.on("send_to_group_ok", (data) => {
  console.log("send_to_group_ok", data);
});

socket.on("send_to_group_error", (err) => {
  console.error("send_to_group_error", err);
});

socket.on("private_message", (data) => {
  console.log("private_message", data);
});

socket.on("group_message", (data) => {
  console.log("group_message", data);
});

socket.on("reply", (msg) => {
  console.log("reply", msg);
});

socket.on("disconnect", (reason) => {
  console.log("disconnect", reason);
});
```
