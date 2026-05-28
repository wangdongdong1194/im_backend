package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"im_backend/internal/app"
	"im_backend/internal/service"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	socketio "github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type bindUserOKResponse struct {
	UserID string `json:"userId"`
}

type bindUserPayload struct {
	UserID string `json:"userId"`
}

type sendToUserPayload struct {
	ToUserID string `json:"toUserId"`
	Message  string `json:"message"`
}

type sendToUserOKResponse struct {
	ToUserID string `json:"toUserId"`
}

type privateMessagePayload struct {
	Message string `json:"message"`
}

type crossNodePrivateMessage struct {
	Event          string `json:"event"`
	ToUserID       string `json:"toUserId"`
	FromUserID     string `json:"fromUserId,omitempty"`
	ConversationID uint   `json:"conversationId,omitempty"`
	MessageID      uint   `json:"messageId,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	Message        string `json:"message"`
}

type sendToGroupPayload struct {
	ConversationID uint   `json:"conversationId"`
	Message        string `json:"message"`
	ClientMsgID    string `json:"clientMsgId"`
}

type groupMessagePayload struct {
	ConversationID uint   `json:"conversationId"`
	FromUserID     string `json:"fromUserId"`
	MessageID      uint   `json:"messageId"`
	Message        string `json:"message"`
	CreatedAt      string `json:"createdAt"`
}

type sendToGroupOKResponse struct {
	ConversationID uint `json:"conversationId"`
	MessageID      uint `json:"messageId"`
	RecipientCount int  `json:"recipientCount"`
	Deduplicated   bool `json:"deduplicated"`
}

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

func parseBindUserPayload(v any) (bindUserPayload, bool) {
	obj, isMap := v.(map[string]any)
	if !isMap {
		return bindUserPayload{}, false
	}

	payload, ok := decodePayload[bindUserPayload](obj)
	if !ok || payload.UserID == "" {
		return bindUserPayload{}, false
	}

	return payload, true
}

func parseSendToUserPayload(v any) (sendToUserPayload, bool) {
	payload, ok := decodePayload[sendToUserPayload](v)
	if !ok || payload.ToUserID == "" || payload.Message == "" {
		return sendToUserPayload{}, false
	}

	return payload, true
}

func parseSendToGroupPayload(v any) (sendToGroupPayload, bool) {
	payload, ok := decodePayload[sendToGroupPayload](v)
	if !ok || payload.ConversationID == 0 || payload.Message == "" {
		return sendToGroupPayload{}, false
	}

	return payload, true
}

func buildPrivateMessageEvent(toUserID string, message string) crossNodePrivateMessage {
	return crossNodePrivateMessage{
		Event:    "private_message",
		ToUserID: toUserID,
		Message:  message,
	}
}

func buildGroupMessageEvent(toUserID string, fromUserID string, conversationID uint, messageID uint, createdAt string, message string) crossNodePrivateMessage {
	return crossNodePrivateMessage{
		Event:          "group_message",
		ToUserID:       toUserID,
		FromUserID:     fromUserID,
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

func publishCrossNodePrivateMessage(application *app.App, event crossNodePrivateMessage) error {
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

			var event crossNodePrivateMessage
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Println("decode cross-node private message failed:", err)
				continue
			}

			targetSocketID, found := socketService.GetSocketIDByUserID(event.ToUserID)
			if !found {
				continue
			}

			target, ok := connections.Get(targetSocketID)
			if !ok {
				continue
			}

			switch event.Event {
			case "group_message":
				target.Emit("group_message", groupMessagePayload{
					ConversationID: event.ConversationID,
					FromUserID:     event.FromUserID,
					MessageID:      event.MessageID,
					Message:        event.Message,
					CreatedAt:      event.CreatedAt,
				})
			default:
				target.Emit("private_message", privateMessagePayload{Message: event.Message})
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
			参数：{ userId: string }
		*/
		cli.On("bind_user", func(args ...any) {
			if len(args) == 0 {
				cli.Emit("bind_user_error", "payload should be object: { userId }")
				return
			}
			payload, ok := parseBindUserPayload(args[0])
			if !ok {
				cli.Emit("bind_user_error", "payload should be object: { userId }")
				return
			}

			if socketService != nil {
				if err := socketService.BindUser(context.Background(), payload.UserID, string(cli.Id())); err != nil {
					if errors.Is(err, service.ErrInvalidUserID) {
						cli.Emit("bind_user_error", "invalid userId")
						return
					}

					log.Println("bind user failed:", err)
					cli.Emit("bind_user_error", "bind user failed")
					return
				}
			}
			log.Println("bind user:", payload.UserID, "socket:", cli.Id())
			cli.Emit("bind_user_ok", bindUserOKResponse{UserID: payload.UserID})
		})

		/*
			发送消息给用户事件
			参数：{ toUserId: string, message: string }
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

			if _, bound := socketService.GetUserIDBySocketID(string(cli.Id())); !bound {
				cli.Emit("send_to_user_error", "bind_user required")
				return
			}

			if len(args) == 0 {
				cli.Emit("send_to_user_error", "missing payload")
				return
			}
			payload, ok := parseSendToUserPayload(args[0])
			if !ok {
				cli.Emit("send_to_user_error", "payload should contain toUserId and message")
				return
			}

			targetSocketID, err := socketService.ResolveTargetSocketID(context.Background(), payload.ToUserID, payload.Message)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrInvalidTargetUser), errors.Is(err, service.ErrInvalidMessage):
					cli.Emit("send_to_user_error", "payload should contain toUserId and message")
				case errors.Is(err, service.ErrTargetUserOffline):
					cli.Emit("send_to_user_error", "target user offline")
				default:
					log.Println("resolve target socket failed:", err)
					cli.Emit("send_to_user_error", "send failed")
				}
				return
			}

			target, found := connections.Get(targetSocketID)
			if found {
				target.Emit("private_message", privateMessagePayload{Message: payload.Message})
				cli.Emit("send_to_user_ok", sendToUserOKResponse{ToUserID: payload.ToUserID})
				return
			}

			if err := publishCrossNodePrivateMessage(application, buildPrivateMessageEvent(payload.ToUserID, payload.Message)); err != nil {
				log.Println("publish cross-node private message failed:", err)
				cli.Emit("send_to_user_error", "send failed")
				return
			}

			cli.Emit("send_to_user_ok", sendToUserOKResponse{ToUserID: payload.ToUserID})
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

			boundUserID, bound := socketService.GetUserIDBySocketID(string(cli.Id()))
			if !bound {
				cli.Emit("send_to_group_error", "bind_user required")
				return
			}

			senderUserID, err := strconv.ParseUint(boundUserID, 10, 64)
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
					target.Emit("group_message", groupMessagePayload{
						ConversationID: payload.ConversationID,
						FromUserID:     boundUserID,
						MessageID:      message.ID,
						Message:        payload.Message,
						CreatedAt:      message.CreatedAt.UTC().Format(time.RFC3339),
					})
					continue
				}

				if publishErr := publishCrossNodePrivateMessage(application, buildGroupMessageEvent(userIDStr, boundUserID, payload.ConversationID, message.ID, message.CreatedAt.UTC().Format(time.RFC3339), payload.Message)); publishErr != nil {
					log.Println("publish cross-node group message failed:", publishErr)
				}
			}

			cli.Emit("send_to_group_ok", sendToGroupOKResponse{
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
