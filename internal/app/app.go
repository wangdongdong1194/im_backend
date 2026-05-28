package app

import (
	"context"
	"fmt"

	"im_backend/config"
	"im_backend/internal/model"
	"im_backend/internal/redisstore"
	"im_backend/internal/repository"
	"im_backend/internal/service"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type App struct {
	Config             config.Config
	MySQLDB            *gorm.DB
	RedisClient        *redis.Client
	SocketBindingStore *redisstore.SocketBindingStore
	SocketService      *service.SocketService
	ChatService        *service.ChatService
	HealthService      *service.HealthService
	UserService        *service.UserService
}

func New(ctx context.Context) (*App, error) {
	cfg := config.Load()

	redisClient := redisstore.NewClient(cfg.RedisAddr, cfg.RedisUsername, cfg.RedisPassword, cfg.RedisDB)
	redisBindingStore := redisstore.NewSocketBindingStore(redisClient, cfg.RedisKeyPrefix)
	if err := redisBindingStore.Ping(ctx); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	var mysqlDB *gorm.DB
	var userService *service.UserService
	var chatService *service.ChatService
	if cfg.MySQLDSN != "" {
		db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
		if err != nil {
			_ = redisBindingStore.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("mysql open failed: %w", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			_ = redisBindingStore.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("mysql db handle failed: %w", err)
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			_ = sqlDB.Close()
			_ = redisBindingStore.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("mysql ping failed: %w", err)
		}

		if err := model.AutoMigrate(db.WithContext(ctx)); err != nil {
			_ = sqlDB.Close()
			_ = redisBindingStore.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("mysql auto migrate failed: %w", err)
		}

		mysqlDB = db
		userRepo := repository.NewGormUserRepository(db)
		friendRequestRepo := repository.NewGormFriendRequestRepository(db)
		friendshipRepo := repository.NewGormFriendshipRepository(db)
		userService = service.NewUserService(userRepo, friendRequestRepo, friendshipRepo)
		conversationRepo := repository.NewGormConversationRepository(db)
		conversationMemberRepo := repository.NewGormConversationMemberRepository(db)
		messageRepo := repository.NewGormMessageRepository(db)
		chatService = service.NewChatService(conversationRepo, conversationMemberRepo, messageRepo)
	}

	return &App{
		Config:             cfg,
		MySQLDB:            mysqlDB,
		RedisClient:        redisClient,
		SocketBindingStore: redisBindingStore,
		SocketService:      service.NewSocketService(redisBindingStore),
		ChatService:        chatService,
		HealthService:      service.NewHealthService(redisClient, cfg.RedisKeyPrefix),
		UserService:        userService,
	}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	if a.MySQLDB != nil {
		sqlDB, err := a.MySQLDB.DB()
		if err != nil {
			return err
		}

		if err := sqlDB.Close(); err != nil {
			return err
		}
	}

	if a.SocketBindingStore == nil {
		return nil
	}

	return a.SocketBindingStore.Close()
}
