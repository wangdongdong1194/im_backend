package service

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidTargetUser = errors.New("invalid target user")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrTargetUserOffline = errors.New("target user offline")
)

type SocketBindingStore interface {
	Bind(ctx context.Context, userID string, socketID string) error
	UnbindBySocketID(ctx context.Context, socketID string) error
	GetSocketIDByUserID(ctx context.Context, userID string) (string, error)
}

type SocketService struct {
	store        SocketBindingStore
	mu           sync.RWMutex
	userToSocket map[string]string
	socketToUser map[string]string
}

func NewSocketService(store SocketBindingStore) *SocketService {
	return &SocketService{
		store:        store,
		userToSocket: make(map[string]string),
		socketToUser: make(map[string]string),
	}
}

func (s *SocketService) BindUser(ctx context.Context, userID string, socketID string) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	s.mu.Lock()
	if oldSocketID, ok := s.userToSocket[userID]; ok && oldSocketID != "" {
		delete(s.socketToUser, oldSocketID)
	}

	s.userToSocket[userID] = socketID
	s.socketToUser[socketID] = userID
	s.mu.Unlock()

	if s.store == nil {
		return nil
	}

	return s.store.Bind(ctx, userID, socketID)
}

func (s *SocketService) UnbindBySocketID(ctx context.Context, socketID string) error {
	s.mu.Lock()
	if userID, ok := s.socketToUser[socketID]; ok {
		delete(s.socketToUser, socketID)
		delete(s.userToSocket, userID)
	}
	s.mu.Unlock()

	if s.store == nil {
		return nil
	}

	return s.store.UnbindBySocketID(ctx, socketID)
}

func (s *SocketService) GetSocketIDByUserID(userID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	socketID, ok := s.userToSocket[userID]
	if !ok || socketID == "" {
		return "", false
	}

	return socketID, true
}

func (s *SocketService) GetUserIDBySocketID(socketID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.socketToUser[socketID]
	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

func (s *SocketService) ResolveTargetSocketID(ctx context.Context, toUserID string, message string) (string, error) {
	if toUserID == "" {
		return "", ErrInvalidTargetUser
	}

	if message == "" {
		return "", ErrInvalidMessage
	}

	socketID, found := s.GetSocketIDByUserID(toUserID)
	if found {
		return socketID, nil
	}

	if s.store == nil {
		return "", ErrTargetUserOffline
	}

	socketID, err := s.store.GetSocketIDByUserID(ctx, toUserID)
	if err != nil || socketID == "" {
		return "", ErrTargetUserOffline
	}

	return socketID, nil
}
