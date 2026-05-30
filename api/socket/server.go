package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"im_backend/internal/app"
	"im_backend/internal/dto"
	"im_backend/internal/service"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	socketio "github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type socketRateState struct {
	lastRefill time.Time
	tokens     float64
}

type socketRateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	states map[string]socketRateState
}

func newSocketRateLimiter(ratePerSecond int, burst int) *socketRateLimiter {
	if ratePerSecond <= 0 {
		ratePerSecond = 40
	}
	if burst <= 0 {
		burst = ratePerSecond
	}

	return &socketRateLimiter{
		rate:   float64(ratePerSecond),
		burst:  float64(burst),
		states: make(map[string]socketRateState),
	}
}

func (l *socketRateLimiter) Allow(key string) bool {
	if l == nil || key == "" {
		return true
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.states[key]
	if !ok {
		l.states[key] = socketRateState{lastRefill: now, tokens: l.burst - 1}
		return true
	}

	elapsed := now.Sub(state.lastRefill).Seconds()
	if elapsed > 0 {
		state.tokens += elapsed * l.rate
		if state.tokens > l.burst {
			state.tokens = l.burst
		}
		state.lastRefill = now
	}

	if state.tokens < 1 {
		l.states[key] = state
		return false
	}

	state.tokens -= 1
	l.states[key] = state
	return true
}

func (l *socketRateLimiter) Remove(key string) {
	if l == nil || key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.states, key)
}

type connectionStore struct {
	mu         sync.RWMutex
	socketByID map[string]*socketio.Socket
}

func newConnectionStore() *connectionStore {
	return &connectionStore{
		socketByID: make(map[string]*socketio.Socket),
	}
}

func (s *connectionStore) Add(socket *socketio.Socket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.socketByID[string(socket.Id())] = socket
}

func (s *connectionStore) Remove(socketID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.socketByID, socketID)
}

func (s *connectionStore) Get(socketID string) (*socketio.Socket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cli, ok := s.socketByID[socketID]
	if !ok || cli == nil {
		return nil, false
	}

	return cli, true
}

func decodePayload[T any](v any) (T, bool) {
	var out T

	b, err := json.Marshal(v)
	if err != nil {
		return out, false
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return out, false
	}

	return out, true
}

func parseBindUserPayload(v any) (dto.BindUserPayload, bool) {
	obj, isMap := v.(map[string]any)
	if !isMap {
		return dto.BindUserPayload{}, false
	}

	payload, ok := decodePayload[dto.BindUserPayload](obj)
	if !ok || payload.ERP == "" || payload.Token == "" {
		return dto.BindUserPayload{}, false
	}

	return payload, true
}

func parseSendToUserPayload(v any) (dto.SendToUserPayload, bool) {
	payload, ok := decodePayload[dto.SendToUserPayload](v)
	if !ok || payload.ToERP == "" || payload.Message == "" {
		return dto.SendToUserPayload{}, false
	}

	return payload, true
}

func parseSendToGroupPayload(v any) (dto.SendToGroupPayload, bool) {
	payload, ok := decodePayload[dto.SendToGroupPayload](v)
	if !ok || payload.ConversationID == 0 || payload.Message == "" {
		return dto.SendToGroupPayload{}, false
	}

	return payload, true
}

func buildPrivateMessageEvent(toERP string, message string) dto.CrossNodePrivateMessage {
	return dto.CrossNodePrivateMessage{
		Event:   "private_message",
		ToERP:   toERP,
		Message: message,
	}
}

func buildGroupMessageEvent(toERP string, fromERP string, conversationID uint, messageID uint, createdAt string, message string) dto.CrossNodePrivateMessage {
	return dto.CrossNodePrivateMessage{
		Event:          "group_message",
		ToERP:          toERP,
		FromERP:        fromERP,
		ConversationID: conversationID,
		MessageID:      messageID,
		CreatedAt:      createdAt,
		Message:        message,
	}
}

func privateMessageChannel(keyPrefix string) string {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = "im_backend"
	}

	return fmt.Sprintf("%s:socket:private_message", prefix)
}

func publishCrossNodePrivateMessage(application *app.App, event dto.CrossNodePrivateMessage) error {
	if application == nil || application.RedisClient == nil {
		return errors.New("redis client not ready")
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return application.RedisClient.Publish(
		context.Background(),
		privateMessageChannel(application.Config.RedisKeyPrefix),
		string(encoded),
	).Err()
}

func startCrossNodePrivateMessageConsumer(application *app.App, socketService *service.SocketService, connections *connectionStore) {
	if application == nil || application.RedisClient == nil || socketService == nil || connections == nil {
		return
	}

	pubsub := application.RedisClient.Subscribe(context.Background(), privateMessageChannel(application.Config.RedisKeyPrefix))
	go func() {
		defer func() {
			if err := pubsub.Close(); err != nil {
				log.Println("close redis pubsub failed:", err)
			}
		}()

		for msg := range pubsub.Channel() {
			if msg == nil || strings.TrimSpace(msg.Payload) == "" {
				continue
			}

			var event dto.CrossNodePrivateMessage
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Println("decode cross-node private message failed:", err)
				continue
			}

			targetSocketID, found := socketService.GetSocketIDByErp(event.ToERP)
			if !found {
				continue
			}

			target, ok := connections.Get(targetSocketID)
			if !ok {
				continue
			}

			switch event.Event {
			case "group_message":
				target.Emit("group_message", dto.GroupMessagePayload{
					ConversationID: event.ConversationID,
					FromERP:        event.FromERP,
					MessageID:      event.MessageID,
					Message:        event.Message,
					CreatedAt:      event.CreatedAt,
				})
			default:
				target.Emit("private_message", dto.PrivateMessagePayload{Message: event.Message})
			}
		}
	}()
}

func parseCORSOrigin(origin string) any {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "*" {
		return "*"
	}

	parts := strings.Split(origin, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		origins = append(origins, part)
	}

	if len(origins) == 0 {
		return "*"
	}

	if len(origins) == 1 {
		return origins[0]
	}

	return origins
}

func NewServer(application *app.App) *socketio.Server {
	opts := socketio.DefaultServerOptions()
	opts.SetCors(&types.Cors{
		Origin:      parseCORSOrigin(application.Config.SocketCORSOrigin),
		Credentials: true,
	})

	server := socketio.NewServer(nil, opts)
	registerEvents(server, application)

	return server
}

func registerEvents(server *socketio.Server, application *app.App) {
	var socketService *service.SocketService
	if application != nil {
		socketService = application.SocketService
	}

	connections := newConnectionStore()
	ratePerSecond := 40
	burst := 80
	if application != nil {
		ratePerSecond = application.Config.SocketSendRate
		burst = application.Config.SocketSendBurst
	}
	limiter := newSocketRateLimiter(ratePerSecond, burst)
	startCrossNodePrivateMessageConsumer(application, socketService, connections)

	server.On("connection", func(clients ...any) {
		if len(clients) == 0 {
			return
		}

		cli, ok := clients[0].(*socketio.Socket)
		if !ok || cli == nil {
			log.Println("connection event with invalid socket payload")
			return
		}

		log.Println("connected:", cli.Id())
		connections.Add(cli)

		/*
			绑定用户事件
			参数：{ erp: string, token: string }
		*/
		cli.On("bind_user", func(args ...any) {
			if !limiter.Allow(string(cli.Id())) {
				cli.Emit("bind_user_error", "rate limit exceeded")
				return
			}

			if len(args) == 0 {
				cli.Emit("bind_user_error", "payload should be object: { erp, token }")
				return
			}
			payload, ok := parseBindUserPayload(args[0])
			if !ok {
				cli.Emit("bind_user_error", "payload should be object: { erp, token }")
				return
			}
			if socketService != nil {
				// 判断token合法性，确保用户只能绑定自己的socket连接，防止冒用他人erp进行恶意操作
				ctx := context.Background()
				token := payload.Token
				cacheErp, err := application.LoginTokenStore.GetTokenUserInfoField(ctx, token, "erp")
				log.Println("cacheErp", cacheErp)
				if err != nil {
					log.Printf("token 校验 redis 异常: %v", err)
					cli.Emit("bind_user_error", "token 校验失败")
					return
				}
				if cacheErp != payload.ERP {
					log.Printf("token 校验失败，token %s 对应的 erp 是 %s，和请求中提供的 erp %s 不匹配", token, cacheErp, payload.ERP)
					cli.Emit("bind_user_error", "token 校验失败")
					return
				}

				if err := socketService.BindUser(context.Background(), payload.ERP, string(cli.Id())); err != nil {
					if errors.Is(err, service.ErrInvalidErp) {
						cli.Emit("bind_user_error", "invalid erp")
						return
					}

					log.Println("bind user failed:", err)
					cli.Emit("bind_user_error", "bind user failed")
					return
				}
			}
			log.Println("bind user:", payload.ERP, "socket:", cli.Id())
			cli.Emit("bind_user_ok", dto.BindUserOKResponse{ERP: payload.ERP})
		})

		/*
			发送消息给用户事件
			参数：{ toERP: string, message: string }
		*/
		cli.On("send_to_user", func(args ...any) {
			if !limiter.Allow(string(cli.Id())) {
				cli.Emit("send_to_user_error", "rate limit exceeded")
				return
			}

			if socketService == nil {
				cli.Emit("send_to_user_error", "socket service not ready")
				return
			}

			if _, bound := socketService.GetErpBySocketID(string(cli.Id())); !bound {
				cli.Emit("send_to_user_error", "bind_user required")
				return
			}

			if len(args) == 0 {
				cli.Emit("send_to_user_error", "missing payload")
				return
			}
			payload, ok := parseSendToUserPayload(args[0])
			if !ok {
				cli.Emit("send_to_user_error", "payload should contain toERP and message")
				return
			}
			
			// 判断ToERP是否存在且为好友 记录到数据库，判断用户是否在线，在线则直接发送，否则通过redis pub/sub分发给其他节点
			ctx:= context.Background()
			selfErp, _ := socketService.GetErpBySocketID(string(cli.Id()))
			isFriend, err := application.LoginTokenStore.IsFriendByErp(ctx, selfErp, payload.ToERP)
			if err != nil {
				log.Printf("判断好友关系失败: %v", err)
				cli.Emit("send_to_user_error", "send failed")
				return
			}
			if !isFriend {
				log.Printf("用户 %s 和 %s 不是好友关系，无法发送消息", selfErp, payload.ToERP)
				cli.Emit("send_to_user_error", "can only send message to friends")
				return
			}

			targetSocketID, err := socketService.ResolveTargetSocketID(context.Background(), payload.ToERP, payload.Message)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrInvalidTargetErp), errors.Is(err, service.ErrInvalidMessage):
					cli.Emit("send_to_user_error", "payload should contain toERP and message")
				case errors.Is(err, service.ErrTargetErpOffline):
					cli.Emit("send_to_user_error", "target user offline")
				default:
					log.Println("resolve target socket failed:", err)
					cli.Emit("send_to_user_error", "send failed")
				}
				return
			}

			target, found := connections.Get(targetSocketID)
			if found {
				target.Emit("private_message", dto.PrivateMessagePayload{Message: payload.Message})
				cli.Emit("send_to_user_ok", dto.SendToUserOKResponse{ToERP: payload.ToERP})
				return
			}

			if err := publishCrossNodePrivateMessage(application, buildPrivateMessageEvent(payload.ToERP, payload.Message)); err != nil {
				log.Println("publish cross-node private message failed:", err)
				cli.Emit("send_to_user_error", "send failed")
				return
			}

			cli.Emit("send_to_user_ok", dto.SendToUserOKResponse{ToERP: payload.ToERP})
		})

		/*
			发送群消息事件
			参数：{ conversationId: number, message: string, clientMsgId?: string }
		*/
		cli.On("send_to_group", func(args ...any) {
			if !limiter.Allow(string(cli.Id())) {
				cli.Emit("send_to_group_error", "rate limit exceeded")
				return
			}

			if len(args) == 0 {
				cli.Emit("send_to_group_error", "missing payload")
				return
			}

			payload, ok := parseSendToGroupPayload(args[0])
			if !ok {
				cli.Emit("send_to_group_error", "payload should contain conversationId and message")
				return
			}

			if socketService == nil || application == nil || application.ChatService == nil {
				cli.Emit("send_to_group_error", "chat service not ready")
				return
			}

			boundErp, bound := socketService.GetErpBySocketID(string(cli.Id()))
			if !bound {
				cli.Emit("send_to_group_error", "bind_user required")
				return
			}

			senderUserID, err := strconv.ParseUint(boundErp, 10, 64)
			if err != nil || senderUserID == 0 {
				cli.Emit("send_to_group_error", "invalid sender userId")
				return
			}

			message, recipients, deduplicated, err := application.ChatService.SendGroupMessage(
				context.Background(),
				payload.ConversationID,
				uint(senderUserID),
				payload.Message,
				payload.ClientMsgID,
			)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrInvalidConversationID), errors.Is(err, service.ErrInvalidMessage), errors.Is(err, service.ErrInvalidSenderID):
					cli.Emit("send_to_group_error", "invalid payload")
				case errors.Is(err, service.ErrConversationNotFound):
					cli.Emit("send_to_group_error", "conversation not found")
				case errors.Is(err, service.ErrConversationTypeInvalid):
					cli.Emit("send_to_group_error", "conversation is not group")
				case errors.Is(err, service.ErrSenderNotInConversation):
					cli.Emit("send_to_group_error", "sender not in conversation")
				default:
					log.Println("send group message failed:", err)
					cli.Emit("send_to_group_error", "send failed")
				}
				return
			}

			for _, recipientUserID := range recipients {
				userIDStr := strconv.FormatUint(uint64(recipientUserID), 10)

				targetSocketID, resolveErr := socketService.ResolveTargetSocketID(context.Background(), userIDStr, payload.Message)
				if resolveErr != nil {
					continue
				}

				target, found := connections.Get(targetSocketID)
				if found {
					target.Emit("group_message", dto.GroupMessagePayload{
						ConversationID: payload.ConversationID,
						FromERP:        boundErp,
						MessageID:      message.ID,
						Message:        payload.Message,
						CreatedAt:      message.CreatedAt.UTC().Format(time.RFC3339),
					})
					continue
				}

				if publishErr := publishCrossNodePrivateMessage(application, buildGroupMessageEvent(userIDStr, boundErp, payload.ConversationID, message.ID, message.CreatedAt.UTC().Format(time.RFC3339), payload.Message)); publishErr != nil {
					log.Println("publish cross-node group message failed:", publishErr)
				}
			}

			cli.Emit("send_to_group_ok", dto.SendToGroupOKResponse{
				ConversationID: payload.ConversationID,
				MessageID:      message.ID,
				RecipientCount: len(recipients),
				Deduplicated:   deduplicated,
			})
		})
		/*
			断开连接事件
		*/
		cli.On("disconnect", func(reason ...any) {
			socketID := string(cli.Id())
			connections.Remove(socketID)
			limiter.Remove(socketID)

			if socketService != nil {
				if err := socketService.UnbindBySocketID(context.Background(), socketID); err != nil {
					log.Println("redis unbind socket failed:", err)
				}
			}
			if len(reason) > 0 {
				log.Println("disconnected:", cli.Id(), "reason:", reason[0])
				return
			}
			log.Println("disconnected:", cli.Id())
		})
		/*
			测试消息事件
		*/
		cli.On("message", func(args ...any) {
			if len(args) == 0 {
				return
			}

			data, ok := args[0].(string)
			if !ok {
				log.Println("recv non-string message")
				return
			}

			log.Println("recv:", data)
			cli.Emit("reply", "ok: "+data)
		})
	})
}
