package controller

import (
	"errors"
	"net/http"
	"strings"

	"im_backend/internal/app"
	"im_backend/internal/dto"
	"im_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	app *app.App
}

func NewUserController(application *app.App) *UserController {
	return &UserController{app: application}
}

// 登录接口
func (ctl *UserController) Login(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	token, user, err := ctl.app.UserService.Login(c.Request.Context(), req.ERP, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrInvalidAccountPayload) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials", "code": 40002})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error(), "code": 40003})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token, User: user})
}

func (ctl *UserController) GetByErp(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready"})
		return
	}

	// 取参数并去除首尾空格
	erpValue := strings.TrimSpace(c.Param("erp"))
	if erpValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user erp"})
		return
	}

	user, err := ctl.app.UserService.GetByERP(c.Request.Context(), erpValue)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// 申请账号
func (ctl *UserController) ApplyAccount(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	var body dto.ApplyAccountBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	ctl.createAccount(c, body, "apply account failed")
}

// Register 接口（对外注册），与 ApplyAccount 行为相同但错误信息不同
func (ctl *UserController) Register(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	var body dto.ApplyAccountBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	ctl.createAccount(c, body, "register account failed")
}

func (ctl *UserController) createAccount(c *gin.Context, body dto.ApplyAccountBody, failedMessage string) {
	user, err := ctl.app.UserService.ApplyAccount(c.Request.Context(), body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserServiceNotReady):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "user service not ready", "code": 40000})
		case errors.Is(err, service.ErrInvalidAccountPayload):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": 40001})
		case errors.Is(err, service.ErrERPAlreadyExists), errors.Is(err, service.ErrUsernameAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"message": err.Error(), "code": 40002})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": failedMessage, "code": 40003})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account created successfully", "code": 20000, "data": user})
}

func (ctl *UserController) ApplyFriendRequest(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	var body dto.ApplyFriendRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	request, err := ctl.app.UserService.ApplyFriendRequest(c.Request.Context(), body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserServiceNotReady):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "user service not ready", "code": 40000})
		case errors.Is(err, service.ErrInvalidFriendERP), errors.Is(err, service.ErrCannotAddSelf):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": 40001})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "user not found", "code": 40004})
		case errors.Is(err, service.ErrAlreadyFriends), errors.Is(err, service.ErrPendingRequestDuplicate):
			c.JSON(http.StatusConflict, gin.H{"message": err.Error(), "code": 40002})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "apply friend request failed", "code": 40003})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "friend request created successfully", "code": 20000, "data": request})
}

func (ctl *UserController) HandleFriendRequest(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	erpValue := strings.TrimSpace(c.Param("erp"))
	if erpValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid friend request erp", "code": 40001})
		return
	}

	var body dto.HandleFriendRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	action := strings.ToLower(strings.TrimSpace(body.Action))
	if action != "accept" && action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid action, only accept or reject is allowed", "code": 40001})
		return
	}

	var err error
	failedMessage := "handle friend request failed"
	if action == "accept" {
		err = ctl.app.UserService.AcceptFriendRequest(c.Request.Context(), erpValue, body)
		failedMessage = "accept friend request failed"
	} else {
		err = ctl.app.UserService.RejectFriendRequest(c.Request.Context(), erpValue, body)
		failedMessage = "reject friend request failed"
	}

	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserServiceNotReady):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "user service not ready", "code": 40000})
		case errors.Is(err, service.ErrInvalidFriendERP), errors.Is(err, service.ErrInvalidFriendRequestID):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": 40001})
		case errors.Is(err, service.ErrFriendRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "friend request not found", "code": 40004})
		case errors.Is(err, service.ErrFriendRequestForbidden):
			c.JSON(http.StatusForbidden, gin.H{"message": "cannot operate this friend request", "code": 40005})
		case errors.Is(err, service.ErrFriendRequestHandled), errors.Is(err, service.ErrAlreadyFriends):
			c.JSON(http.StatusConflict, gin.H{"message": err.Error(), "code": 40002})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": failedMessage, "code": 40003})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok", "code": 20000})
}
