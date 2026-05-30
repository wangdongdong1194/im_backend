package service

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrInvalidErp       = errors.New("invalid erp")
	ErrInvalidTargetErp = errors.New("invalid target erp")
	ErrInvalidMessage   = errors.New("invalid message")
	ErrTargetErpOffline = errors.New("target erp offline")
)

type SocketBindingStore interface {
	Bind(ctx context.Context, erp string, socketID string) error
	UnbindBySocketID(ctx context.Context, socketID string) error
	GetSocketIDByErp(ctx context.Context, erp string) (string, error)
}

type SocketService struct {
	store       SocketBindingStore
	mu          sync.RWMutex
	erpToSocket map[string]string
	socketToErp map[string]string
}

func NewSocketService(store SocketBindingStore) *SocketService {
	return &SocketService{
		store:       store,
		erpToSocket: make(map[string]string),
		socketToErp: make(map[string]string),
	}
}

func (s *SocketService) BindUser(ctx context.Context, erp string, socketID string) error {
	if erp == "" {
		return ErrInvalidErp
	}

	s.mu.Lock()
	if oldSocketID, ok := s.erpToSocket[erp]; ok && oldSocketID != "" {
		delete(s.socketToErp, oldSocketID)
	}

	s.erpToSocket[erp] = socketID
	s.socketToErp[socketID] = erp
	s.mu.Unlock()

	if s.store == nil {
		return nil
	}

	return s.store.Bind(ctx, erp, socketID)
}

func (s *SocketService) UnbindBySocketID(ctx context.Context, socketID string) error {
	s.mu.Lock()
	if erp, ok := s.socketToErp[socketID]; ok {
		delete(s.socketToErp, socketID)
		delete(s.erpToSocket, erp)
	}
	s.mu.Unlock()

	if s.store == nil {
		return nil
	}

	return s.store.UnbindBySocketID(ctx, socketID)
}

func (s *SocketService) GetSocketIDByErp(erp string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	socketID, ok := s.erpToSocket[erp]
	if !ok || socketID == "" {
		return "", false
	}

	return socketID, true
}

func (s *SocketService) GetErpBySocketID(socketID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	erp, ok := s.socketToErp[socketID]
	if !ok || erp == "" {
		return "", false
	}

	return erp, true
}

func (s *SocketService) ResolveTargetSocketID(ctx context.Context, toErp string, message string) (string, error) {
	if toErp == "" {
		return "", ErrInvalidTargetErp
	}

	if message == "" {
		return "", ErrInvalidMessage
	}

	socketID, found := s.GetSocketIDByErp(toErp)
	if found {
		return socketID, nil
	}

	if s.store == nil {
		return "", ErrTargetErpOffline
	}

	socketID, err := s.store.GetSocketIDByErp(ctx, toErp)
	if err != nil || socketID == "" {
		return "", ErrTargetErpOffline
	}

	return socketID, nil
}
