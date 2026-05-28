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

	r.GET("/health", healthController.Health)
	r.GET("/mysql/test-read", mySQLController.TestRead)
	r.GET("/users/:id", userController.GetByID)
	r.POST("/accounts/apply", userController.ApplyAccount)
	r.POST("/friend-requests/apply", userController.ApplyFriendRequest)
	r.POST("/friend-requests/:id/accept", userController.AcceptFriendRequest)
	r.POST("/friend-requests/:id/reject", userController.RejectFriendRequest)
	r.GET("/conversations/:id/messages", chatController.ListConversationMessages)

	r.Any("/socket.io/*any", gin.WrapH(socketHandler))

	return r
}
