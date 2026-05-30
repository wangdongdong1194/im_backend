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
	// Friend list cache
	GetFriends(ctx context.Context, erp string) ([]string, error)
	SetFriends(ctx context.Context, erp string, friends []string, expire time.Duration) error
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

	// 更新完 DB 后，异步刷新 Redis 中双方的好友列表缓存（best-effort）
	if s.loginTokenStore != nil {
		fromUser, _ := s.userRepo.GetByID(ctx, request.FromUserID)
		toUser, _ := s.userRepo.GetByID(ctx, request.ToUserID)
		fromErp := ""
		toErp := ""
		if fromUser != nil {
			fromErp = fromUser.Erp
		}
		if toUser != nil {
			toErp = toUser.Erp
		}

		go func(fromUserID, toUserID uint, fromErp, toErp string) {
			// 列出 fromUser 的好友 erp
			fromItems, err := s.friendshipRepo.ListByUser(context.Background(), repository.ListFriendshipsOption{UserID: fromUserID, Offset: 0, Limit: 200})
			if err == nil {
				fromErps := make([]string, 0, len(fromItems))
				for _, it := range fromItems {
					if fu, err := s.userRepo.GetByID(context.Background(), it.FriendID); err == nil {
						fromErps = append(fromErps, fu.Erp)
					}
				}
				_ = s.loginTokenStore.SetFriends(context.Background(), fromErp, fromErps, time.Hour)
			}

			// 列出 toUser 的好友 erp
			toItems, err := s.friendshipRepo.ListByUser(context.Background(), repository.ListFriendshipsOption{UserID: toUserID, Offset: 0, Limit: 200})
			if err == nil {
				toErps := make([]string, 0, len(toItems))
				for _, it := range toItems {
					if tu, err := s.userRepo.GetByID(context.Background(), it.FriendID); err == nil {
						toErps = append(toErps, tu.Erp)
					}
				}
				_ = s.loginTokenStore.SetFriends(context.Background(), toErp, toErps, time.Hour)
			}
		}(request.FromUserID, request.ToUserID, fromErp, toErp)
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
		Email:    "",
		Bio:      "",
	}
	if user.Nickname != nil {
		info.Nickname = *user.Nickname
	}
	if user.Email != nil {
		info.Email = *user.Email
	}
	if user.Bio != nil {
		info.Bio = *user.Bio
	}
	values := map[string]interface{}{
		"erp":      info.ERP,
		"username": info.Username,
		"nickname": info.Nickname,
		"phone":    info.Phone,
		"email":    info.Email,
		"bio":      info.Bio,
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

// Logout 删除 token 相关信息
func (s *UserService) Logout(ctx context.Context, token string) error {
	if s == nil || s.loginTokenStore == nil {
		return ErrUserServiceNotReady
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("invalid token")
	}

	info, err := s.loginTokenStore.GetTokenUserInfo(ctx, token)
	if err == nil {
		if erp, ok := info["erp"]; ok && erp != "" {
			_ = s.loginTokenStore.DelErpToken(ctx, erp)
		}
	}

	if err := s.loginTokenStore.DelTokenUserInfo(ctx, token); err != nil {
		return err
	}

	return nil
}

// ListFriends 列出用户的好友列表
func (s *UserService) ListFriends(ctx context.Context, erp string, offset int, limit int) ([]dto.UserInfo, error) {
	if s == nil || s.userRepo == nil || s.friendshipRepo == nil {
		return nil, ErrUserServiceNotReady
	}
	erp = strings.TrimSpace(erp)
	if erp == "" {
		return nil, ErrInvalidFriendERP
	}

	// 1) try redis cache first
	if s.loginTokenStore != nil {
		if cached, err := s.loginTokenStore.GetFriends(ctx, erp); err == nil && len(cached) > 0 {
			results := make([]dto.UserInfo, 0, len(cached))
			for _, friendErp := range cached {
				friendUser, err := s.userRepo.GetByERP(ctx, friendErp)
				if err != nil {
					continue
				}
				ui := dto.UserInfo{
					ERP:      friendUser.Erp,
					Username: friendUser.Username,
					Nickname: "",
					Phone:    friendUser.Phone,
					Email:    "",
				}
				if friendUser.Nickname != nil {
					ui.Nickname = *friendUser.Nickname
				}
				if friendUser.Email != nil {
					ui.Email = *friendUser.Email
				}
				results = append(results, ui)
			}
			return results, nil
		}
	}

	// 2) fallback to MySQL
	user, err := s.userRepo.GetByERP(ctx, erp)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	items, err := s.friendshipRepo.ListByUser(ctx, repository.ListFriendshipsOption{UserID: user.ID, Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}

	results := make([]dto.UserInfo, 0, len(items))
	friendErps := make([]string, 0, len(items))
	for _, f := range items {
		friendUser, err := s.userRepo.GetByID(ctx, f.FriendID)
		if err != nil {
			// skip missing friend record
			continue
		}
		ui := dto.UserInfo{
			ERP:      friendUser.Erp,
			Username: friendUser.Username,
			Nickname: "",
			Phone:    friendUser.Phone,
			Email:    "",
			Bio:      "",
		}
		if friendUser.Nickname != nil {
			ui.Nickname = *friendUser.Nickname
		}
		if friendUser.Email != nil {
			ui.Email = *friendUser.Email
		}
		if friendUser.Bio != nil {
			ui.Bio = *friendUser.Bio
		}
		results = append(results, ui)
		friendErps = append(friendErps, friendUser.Erp)
	}

	// cache to redis asynchronously (best-effort)
	if s.loginTokenStore != nil && len(friendErps) > 0 {
		go func(erps []string) {
			_ = s.loginTokenStore.SetFriends(context.Background(), erp, erps, time.Hour)
		}(friendErps)
	}

	return results, nil
}
