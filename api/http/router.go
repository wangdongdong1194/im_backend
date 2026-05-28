package httpapi

import (
	"net/http"

	"im_backend/api/http/controller"
	"im_backend/internal/app"

	"github.com/gin-gonic/gin"
)

func NewRouter(application *app.App, socketHandler http.Handler) *gin.Engine {
	healthController := controller.NewHealthController(application)
	mySQLController := controller.NewMySQLController(application)
	userController := controller.NewUserController(application)
	chatController := controller.NewChatController(application)

	r := gin.Default()

	r.GET("/health", healthController.Health) // 健康检查接口
	r.GET("/mysql/test-read", mySQLController.TestRead)
	r.GET("/users/:erp", userController.GetByErp)                                 // 获取用户信息接口
	r.POST("/accounts/apply", userController.ApplyAccount)                        // 申请账号接口
	r.POST("/friend-requests/apply", userController.ApplyFriendRequest)           // 申请好友接口
	r.POST("/friend-requests/:erp/handle", userController.HandleFriendRequest)    // 处理好友请求接口-通过/拒绝
	r.GET("/conversations/:id/messages", chatController.ListConversationMessages) // 获取会话消息接口

	r.Any("/socket.io/*any", gin.WrapH(socketHandler))

	return r
}
