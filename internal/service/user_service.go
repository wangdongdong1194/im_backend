package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"im_backend/internal/model"
	"im_backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrUserServiceNotReady     = errors.New("user service not ready")
	ErrInvalidFriendERP        = errors.New("invalid erp")
	ErrInvalidFriendRequestID  = errors.New("invalid friend request id")
	ErrFriendRequestNotFound   = errors.New("friend request not found")
	ErrFriendRequestForbidden  = errors.New("friend request forbidden")
	ErrFriendRequestHandled    = errors.New("friend request already handled")
	ErrCannotAddSelf           = errors.New("cannot add self as friend")
	ErrAlreadyFriends          = errors.New("already friends")
	ErrPendingRequestDuplicate = errors.New("pending friend request already exists")
	ErrInvalidAccountPayload   = errors.New("invalid account payload")
	ErrERPAlreadyExists        = errors.New("erp already exists")
	ErrUsernameAlreadyExists   = errors.New("username already exists")
)

type UserService struct {
	userRepo          repository.UserRepository
	friendRequestRepo repository.FriendRequestRepository
	friendshipRepo    repository.FriendshipRepository
}

func NewUserService(
	userRepo repository.UserRepository,
	friendRequestRepo repository.FriendRequestRepository,
	friendshipRepo repository.FriendshipRepository,
) *UserService {
	return &UserService{
		userRepo:          userRepo,
		friendRequestRepo: friendRequestRepo,
		friendshipRepo:    friendshipRepo,
	}
}

func (s *UserService) GetByERP(ctx context.Context, erp string) (*model.User, error) {
	if s == nil || s.userRepo == nil {
		return nil, ErrUserServiceNotReady
	}
	if erp == "" {
		return nil, ErrInvalidFriendERP
	}

	user, err := s.userRepo.GetByERP(ctx, erp)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		if errors.Is(err, repository.ErrRepositoryNotReady) {
			return nil, ErrUserServiceNotReady
		}
		return nil, err
	}

	return user, nil
}

func (s *UserService) ApplyAccount(
	ctx context.Context,
	erp string,
	username string,
	nickname string,
	password string,
) (*model.User, error) {
	if s == nil || s.userRepo == nil {
		return nil, ErrUserServiceNotReady
	}

	erp = strings.TrimSpace(erp)
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	password = strings.TrimSpace(password)
	if erp == "" || password == "" {
		return nil, ErrInvalidAccountPayload
	}
	if username == "" {
		username = erp
	}
	if nickname == "" {
		nickname = username
	}

	if _, err := s.userRepo.GetByERP(ctx, erp); err == nil {
		return nil, ErrERPAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, ErrUsernameAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	usernameValue := username
	nicknameValue := nickname

	user := &model.User{
		Erp:          erp,
		Username:     &usernameValue,
		Nickname:     &nicknameValue,
		Email:        nil,
		Phone:        nil,
		PasswordHash: string(hashBytes),
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) ApplyFriendRequest(ctx context.Context, fromERP string, toERP string, reason string) (*model.FriendRequest, error) {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil || s.friendshipRepo == nil {
		return nil, ErrUserServiceNotReady
	}
	fromERP = strings.TrimSpace(fromERP)
	toERP = strings.TrimSpace(toERP)
	if fromERP == "" || toERP == "" {
		return nil, ErrInvalidFriendERP
	}

	fromUser, err := s.userRepo.GetByUserID(ctx, fromERP)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	toUser, err := s.userRepo.GetByUserID(ctx, toERP)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if fromUser.ID == toUser.ID {
		return nil, ErrCannotAddSelf
	}

	if _, err := s.friendshipRepo.GetByPair(ctx, fromUser.ID, toUser.ID); err == nil {
		return nil, ErrAlreadyFriends
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.friendshipRepo.GetByPair(ctx, toUser.ID, fromUser.ID); err == nil {
		return nil, ErrAlreadyFriends
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	pendingRequests, err := s.friendRequestRepo.ListByToUser(ctx, repository.ListFriendRequestsOption{
		ToUserID: fromUser.ID,
		Status:   "pending",
		Limit:    200,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range pendingRequests {
		if item.FromUserID == toUser.ID {
			return nil, ErrPendingRequestDuplicate
		}
	}

	request := &model.FriendRequest{
		FromUserID:  fromUser.ID,
		ToUserID:    toUser.ID,
		ApplyReason: reason,
		Status:      "pending",
	}

	if err := s.friendRequestRepo.Create(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

func (s *UserService) AcceptFriendRequest(ctx context.Context, erp string, operatorERP string, remark string) error {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil || s.friendshipRepo == nil {
		return ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	if erp == "" {
		return ErrInvalidFriendERP
	}
	operatorERP = strings.TrimSpace(operatorERP)
	if operatorERP == "" {
		return ErrInvalidFriendERP
	}

	request, err := s.findPendingFriendRequestByERP(ctx, erp, operatorERP)
	if err != nil {
		return err
	}

	if _, err := s.friendshipRepo.GetByPair(ctx, request.FromUserID, request.ToUserID); err == nil {
		return ErrAlreadyFriends
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	if err := s.friendshipRepo.Create(ctx, &model.Friendship{
		UserID:   request.FromUserID,
		FriendID: request.ToUserID,
		Remark:   "",
		Status:   "active",
	}); err != nil {
		return err
	}

	if err := s.friendshipRepo.Create(ctx, &model.Friendship{
		UserID:   request.ToUserID,
		FriendID: request.FromUserID,
		Remark:   remark,
		Status:   "active",
	}); err != nil {
		return err
	}

	if err := s.friendRequestRepo.UpdateByID(ctx, request.ID, map[string]any{
		"status":     "accepted",
		"handled_at": now,
	}); err != nil {
		return err
	}

	return nil
}

func (s *UserService) RejectFriendRequest(ctx context.Context, erp string, operatorERP string) error {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil {
		return ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	if erp == "" {
		return ErrInvalidFriendERP
	}
	operatorERP = strings.TrimSpace(operatorERP)
	if operatorERP == "" {
		return ErrInvalidFriendERP
	}

	request, err := s.findPendingFriendRequestByERP(ctx, erp, operatorERP)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := s.friendRequestRepo.UpdateByID(ctx, request.ID, map[string]any{
		"status":     "rejected",
		"handled_at": now,
	}); err != nil {
		return err
	}

	return nil
}

func (s *UserService) findPendingFriendRequestByERP(ctx context.Context, fromERP string, toERP string) (*model.FriendRequest, error) {
	fromUser, err := s.userRepo.GetByERP(ctx, fromERP)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	toUser, err := s.userRepo.GetByERP(ctx, toERP)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	pendingRequests, err := s.friendRequestRepo.ListByToUser(ctx, repository.ListFriendRequestsOption{
		ToUserID: toUser.ID,
		Status:   "pending",
		Limit:    200,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range pendingRequests {
		if item.FromUserID == fromUser.ID {
			req := item
			return &req, nil
		}
	}

	return nil, ErrFriendRequestNotFound
}
