package controller

import (
	"net/http"

	"im_backend/internal/app"

	"github.com/gin-gonic/gin"
)

type HealthController struct {
	app *app.App
}

func NewHealthController(application *app.App) *HealthController {
	return &HealthController{app: application}
}

func (ctl *HealthController) Health(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.HealthService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "health service not ready"})
		return
	}

	if err := ctl.app.HealthService.MarkHealthy(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
