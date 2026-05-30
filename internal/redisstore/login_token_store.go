package redisstore

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type LoginTokenStore struct {
	client *redis.Client
	prefix string // 不含末尾 ':'，为空表示不使用前缀
}

func NewLoginTokenStore(client *redis.Client, prefix string) *LoginTokenStore {
	p := strings.TrimSpace(prefix)
	// do not append colon here; methods will handle it
	return &LoginTokenStore{client: client, prefix: p}
}

func (s *LoginTokenStore) tokenUserInfoKey(token string) string {
	base := "login:token:userinfo:"
	if s == nil || s.prefix == "" {
		return base + token
	}
	return s.prefix + ":" + base + token
}

func (s *LoginTokenStore) erpTokenKey(erp string) string {
	base := "login:erp:token:"
	if s == nil || s.prefix == "" {
		return base + erp
	}
	return s.prefix + ":" + base + erp
}

func (s *LoginTokenStore) friendKey(erp string) string {
	base := "friend:"
	if s == nil || s.prefix == "" {
		return base + erp
	}
	return s.prefix + ":" + base + erp
}

func (s *LoginTokenStore) SetTokenUserInfo(ctx context.Context, token string, fields map[string]interface{}, expire time.Duration) error {
	key := s.tokenUserInfoKey(token)
	if err := s.client.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, key, expire).Err()
}

func (s *LoginTokenStore) GetTokenUserInfo(ctx context.Context, token string) (map[string]string, error) {
	key := s.tokenUserInfoKey(token)
	return s.client.HGetAll(ctx, key).Result()
}

func (s *LoginTokenStore) GetTokenUserInfoField(ctx context.Context, token string, field string) (string, error) {
	key := s.tokenUserInfoKey(token)
	return s.client.HGet(ctx, key, field).Result()
}

func (s *LoginTokenStore) DelTokenUserInfo(ctx context.Context, token string) error {
	key := s.tokenUserInfoKey(token)
	return s.client.Del(ctx, key).Err()
}

func (s *LoginTokenStore) SetErpToken(ctx context.Context, erp string, token string, expire time.Duration) error {
	key := s.erpTokenKey(erp)
	return s.client.Set(ctx, key, token, expire).Err()
}

func (s *LoginTokenStore) GetErpToken(ctx context.Context, erp string) (string, error) {
	key := s.erpTokenKey(erp)
	return s.client.Get(ctx, key).Result()
}

func (s *LoginTokenStore) DelErpToken(ctx context.Context, erp string) error {
	key := s.erpTokenKey(erp)
	return s.client.Del(ctx, key).Err()
}

// GetFriends 从 redis 中读取某用户的好友列表（set），若不存在返回空切片且不报错
func (s *LoginTokenStore) GetFriends(ctx context.Context, erp string) ([]string, error) {
	key := s.friendKey(erp)
	members, err := s.client.SMembers(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, err
	}
	return members, nil
}

// SetFriends 将好友列表写入 redis set 并设置过期时间
func (s *LoginTokenStore) SetFriends(ctx context.Context, erp string, friends []string, expire time.Duration) error {
	key := s.friendKey(erp)
	if len(friends) == 0 {
		// ensure key removed when empty
		return s.client.Del(ctx, key).Err()
	}
	// Use SAdd to add multiple members
	members := make([]interface{}, 0, len(friends))
	for _, f := range friends {
		members = append(members, f)
	}
	if err := s.client.SAdd(ctx, key, members...).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, key, expire).Err()
}

// 判断两个用户是否为好友关系，判断逻辑：将两个用户的erp拼接成一个字符串（按照字典序升序排序），查询redis中是否存在key "prefix:friend:拼接字符串"，存在则为好友关系
func (s *LoginTokenStore) IsFriendByErp(ctx context.Context, erp1, erp2 string) (bool, error) {
	var newStr string
	// 字典序升序排序
	if strings.Compare(erp1, erp2) <= 0 {
		newStr = erp1 + erp2
	} else {
		newStr = erp2 + erp1
	}
	key := s.friendKey(newStr)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
