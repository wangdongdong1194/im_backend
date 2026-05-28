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
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "chat service not ready"})
		return
	}

	conversationID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || conversationID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	beforeID := uint64(0)
	beforeRaw := ctx.Query("beforeId")
	if beforeRaw != "" {
		parsed, parseErr := strconv.ParseUint(beforeRaw, 10, 64)
		if parseErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid beforeId"})
			return
		}
		beforeID = parsed
	}

	offset, err := parseIntWithDefault(ctx.Query("offset"), 0)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	limit, err := parseIntWithDefault(ctx.Query("limit"), 20)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
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
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		case errors.Is(err, service.ErrChatServiceNotReady):
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "chat service not ready"})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query messages failed"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"items": messages})
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
