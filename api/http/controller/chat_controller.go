package controller

import (
	"errors"
	"net/http"
	"strconv"

	"im_backend/internal/app"
	"im_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ChatController struct {
	application *app.App
}

func NewChatController(application *app.App) *ChatController {
	return &ChatController{application: application}
}

func (c *ChatController) ListConversationMessages(ctx *gin.Context) {
	if c == nil || c.application == nil || c.application.ChatService == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"message": "chat service not ready", "code": 40000})
		return
	}

	conversationID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || conversationID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid conversation id", "code": 40001})
		return
	}

	beforeID := uint64(0)
	beforeRaw := ctx.Query("beforeId")
	if beforeRaw != "" {
		parsed, parseErr := strconv.ParseUint(beforeRaw, 10, 64)
		if parseErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid beforeId", "code": 40001})
			return
		}
		beforeID = parsed
	}

	offset, err := parseIntWithDefault(ctx.Query("offset"), 0)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid offset", "code": 40001})
		return
	}

	limit, err := parseIntWithDefault(ctx.Query("limit"), 20)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid limit", "code": 40001})
		return
	}

	messages, err := c.application.ChatService.ListConversationMessages(
		ctx.Request.Context(),
		uint(conversationID),
		uint(beforeID),
		offset,
		limit,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidConversationID):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid conversation id", "code": 40001})
		case errors.Is(err, service.ErrChatServiceNotReady):
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"message": "chat service not ready", "code": 40000})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "query messages failed", "code": 40003})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "ok", "code": 20000, "data": messages})
}

func parseIntWithDefault(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}

	return value, nil
}
