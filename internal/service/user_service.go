package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"im_backend/internal/dto"
	"im_backend/internal/model"
	"im_backend/internal/repository"

	"github.com/google/uuid"
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

// LoginTokenStore interface
// 用于存储和获取登录 token 相关信息
// 实现应在 internal/redisstore/login_token_store.go
type LoginTokenStore interface {
	SetTokenUserInfo(ctx context.Context, token string, fields map[string]interface{}, expire time.Duration) error
	GetTokenUserInfo(ctx context.Context, token string) (map[string]string, error)
	DelTokenUserInfo(ctx context.Context, token string) error

	SetErpToken(ctx context.Context, erp string, token string, expire time.Duration) error
	GetErpToken(ctx context.Context, erp string) (string, error)
	DelErpToken(ctx context.Context, erp string) error
}

type UserService struct {
	userRepo          repository.UserRepository
	friendRequestRepo repository.FriendRequestRepository
	friendshipRepo    repository.FriendshipRepository
	loginTokenStore   LoginTokenStore
}

func NewUserService(
	userRepo repository.UserRepository,
	friendRequestRepo repository.FriendRequestRepository,
	friendshipRepo repository.FriendshipRepository,
	loginTokenStore LoginTokenStore,
) *UserService {
	return &UserService{
		userRepo:          userRepo,
		friendRequestRepo: friendRequestRepo,
		friendshipRepo:    friendshipRepo,
		loginTokenStore:   loginTokenStore,
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
	body dto.ApplyAccountBody,
) (*model.User, error) {
	if s == nil || s.userRepo == nil {
		return nil, ErrUserServiceNotReady
	}

	erp := body.ERP
	password := body.Password
	username := body.Username
	phone := body.Phone
	nickname := ""
	if body.Nickname != nil {
		nickname = *body.Nickname
	}

	erp = strings.TrimSpace(erp)
	username = strings.TrimSpace(username)
	phone = strings.TrimSpace(phone)
	nickname = strings.TrimSpace(nickname)
	password = strings.TrimSpace(password)
	if erp == "" || password == "" || username == "" || phone == "" {
		return nil, ErrInvalidAccountPayload
	}
	if nickname == "" {
		nickname = username
	}

	if _, err := s.userRepo.GetByERP(ctx, erp); err == nil {
		return nil, ErrERPAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Erp:          erp,
		Username:     username,
		Nickname:     &nickname,
		Email:        nil,
		Phone:        phone,
		PasswordHash: string(hashBytes),
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) ApplyFriendRequest(ctx context.Context, body dto.ApplyFriendRequestBody) (*model.FriendRequest, error) {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil || s.friendshipRepo == nil {
		return nil, ErrUserServiceNotReady
	}
	fromERP := body.FromERP
	toERP := body.ToERP
	reason := body.ApplyReason
	fromERP = strings.TrimSpace(fromERP)
	toERP = strings.TrimSpace(toERP)
	reason = strings.TrimSpace(reason)
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

func (s *UserService) AcceptFriendRequest(ctx context.Context, erp string, body dto.HandleFriendRequestBody) error {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil || s.friendshipRepo == nil {
		return ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	if erp == "" {
		return ErrInvalidFriendERP
	}
	operatorERP := body.OperatorERP
	remark := body.Remark
	operatorERP = strings.TrimSpace(operatorERP)
	remark = strings.TrimSpace(remark)
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

func (s *UserService) RejectFriendRequest(ctx context.Context, erp string, body dto.HandleFriendRequestBody) error {
	if s == nil || s.userRepo == nil || s.friendRequestRepo == nil {
		return ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	if erp == "" {
		return ErrInvalidFriendERP
	}
	operatorERP := body.OperatorERP
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

// Login 登录接口，校验密码，生成 token，存 redis
func (s *UserService) Login(ctx context.Context, erp string, password string) (string, dto.UserInfo, error) {
	if s == nil || s.userRepo == nil || s.loginTokenStore == nil {
		return "", dto.UserInfo{}, ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	password = strings.TrimSpace(password)
	if erp == "" || password == "" {
		return "", dto.UserInfo{}, ErrInvalidAccountPayload
	}
	user, err := s.userRepo.GetByERP(ctx, erp)
	if err != nil {
		return "", dto.UserInfo{}, ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", dto.UserInfo{}, ErrInvalidAccountPayload
	}
	token := strings.ToLower(uuid.NewString())
	info := dto.UserInfo{
		ERP:      user.Erp,
		Username: user.Username,
		Nickname: "",
		Phone:    user.Phone,
	}
	if user.Nickname != nil {
		info.Nickname = *user.Nickname
	}
	values := map[string]interface{}{
		"erp":      info.ERP,
		"username": info.Username,
		"nickname": info.Nickname,
		"phone":    info.Phone,
	}
	// 若存在token且未过期，则拒绝登录，要求先登出旧设备（删除旧token），避免重复登录导致数据不一致等问题
	oldToken, _ := s.loginTokenStore.GetErpToken(ctx, user.Erp)
	log.Println("oldToken:", oldToken)
	result, err := s.loginTokenStore.GetTokenUserInfo(ctx, oldToken)
	log.Println("redis result:", result, "err:", err)
	if oldToken != "" {
		s.loginTokenStore.DelTokenUserInfo(ctx, oldToken)
	}

	expire := time.Duration(13+time.Now().UnixNano()%11) * time.Hour // 13~23h
	if err := s.loginTokenStore.SetTokenUserInfo(ctx, token, values, expire); err != nil {
		return "", dto.UserInfo{}, err
	}
	if err := s.loginTokenStore.SetErpToken(ctx, info.ERP, token, expire); err != nil {
		return "", dto.UserInfo{}, err
	}
	return token, info, nil
}
