package redisstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type SocketBindingStore struct {
	client    *redis.Client
	keyPrefix string
}

func NewClient(addr string, username string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})
}

func NewSocketBindingStore(client *redis.Client, keyPrefix string) *SocketBindingStore {
	return &SocketBindingStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

func (s *SocketBindingStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}

	return s.client.Ping(ctx).Err()
}

func (s *SocketBindingStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Close()
}

func (s *SocketBindingStore) Bind(ctx context.Context, userID string, socketID string) error {
	oldSocketID, err := s.client.Get(ctx, s.userKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	pipe := s.client.TxPipeline()
	if oldSocketID != "" {
		pipe.Del(ctx, s.socketKey(oldSocketID))
	}
	pipe.Set(ctx, s.userKey(userID), socketID, 0)
	pipe.Set(ctx, s.socketKey(socketID), userID, 0)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SocketBindingStore) UnbindBySocketID(ctx context.Context, socketID string) error {
	userID, err := s.client.Get(ctx, s.socketKey(socketID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, s.socketKey(socketID))
	pipe.Del(ctx, s.userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SocketBindingStore) GetSocketIDByUserID(ctx context.Context, userID string) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("redis client is nil")
	}

	socketID, err := s.client.Get(ctx, s.userKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return socketID, nil
}

func (s *SocketBindingStore) userKey(userID string) string {
	return fmt.Sprintf("%s:socket:user:%s", s.keyPrefix, userID)
}

func (s *SocketBindingStore) socketKey(socketID string) string {
	return fmt.Sprintf("%s:socket:conn:%s", s.keyPrefix, socketID)
}
