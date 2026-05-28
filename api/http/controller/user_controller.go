package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"im_backend/internal/app"
	"im_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	app *app.App
}

type applyAccountBody struct {
	ERP      string `json:"erp"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

type applyFriendRequestBody struct {
	FromERP     string `json:"fromErp"`
	ToERP       string `json:"toErp"`
	ApplyReason string `json:"applyReason"`
}

type handleFriendRequestBody struct {
	OperatorERP string `json:"operatorErp"`
	Remark      string `json:"remark"`
}

func NewUserController(application *app.App) *UserController {
	return &UserController{app: application}
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

	var body applyAccountBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	ctl.createAccount(c, body.ERP, body.Username, body.Nickname, body.Password, "apply account failed")
}

func (ctl *UserController) createAccount(c *gin.Context, erp string, username string, nickname string, password string, failedMessage string) {
	user, err := ctl.app.UserService.ApplyAccount(c.Request.Context(), erp, username, nickname, password)
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

	var body applyFriendRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	request, err := ctl.app.UserService.ApplyFriendRequest(c.Request.Context(), body.FromERP, body.ToERP, body.ApplyReason)
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

func (ctl *UserController) AcceptFriendRequest(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	idValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid friend request id", "code": 40001})
		return
	}

	var body handleFriendRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	err = ctl.app.UserService.AcceptFriendRequest(c.Request.Context(), uint(idValue), body.OperatorERP, body.Remark)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserServiceNotReady):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "user service not ready", "code": 40000})
		case errors.Is(err, service.ErrInvalidFriendRequestID), errors.Is(err, service.ErrInvalidFriendERP):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": 40001})
		case errors.Is(err, service.ErrFriendRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "friend request not found", "code": 40004})
		case errors.Is(err, service.ErrFriendRequestForbidden):
			c.JSON(http.StatusForbidden, gin.H{"message": "cannot operate this friend request", "code": 40005})
		case errors.Is(err, service.ErrFriendRequestHandled), errors.Is(err, service.ErrAlreadyFriends):
			c.JSON(http.StatusConflict, gin.H{"message": err.Error(), "code": 40002})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "accept friend request failed", "code": 40003})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok", "code": 20000})
}

func (ctl *UserController) RejectFriendRequest(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.UserService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "user service not ready", "code": 40000})
		return
	}

	idValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || idValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid friend request id", "code": 40001})
		return
	}

	var body handleFriendRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "code": 40001})
		return
	}

	err = ctl.app.UserService.RejectFriendRequest(c.Request.Context(), uint(idValue), body.OperatorERP)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserServiceNotReady):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "user service not ready", "code": 40000})
		case errors.Is(err, service.ErrInvalidFriendRequestID), errors.Is(err, service.ErrInvalidFriendERP):
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": 40001})
		case errors.Is(err, service.ErrFriendRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "friend request not found", "code": 40004})
		case errors.Is(err, service.ErrFriendRequestForbidden):
			c.JSON(http.StatusForbidden, gin.H{"message": "cannot operate this friend request", "code": 40005})
		case errors.Is(err, service.ErrFriendRequestHandled):
			c.JSON(http.StatusConflict, gin.H{"message": err.Error(), "code": 40002})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "reject friend request failed", "code": 40003})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok", "code": 20000})
}
