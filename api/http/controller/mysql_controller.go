package controller

import (
	"net/http"

	"im_backend/internal/app"

	"github.com/gin-gonic/gin"
)

type MySQLController struct {
	app *app.App
}

func NewMySQLController(application *app.App) *MySQLController {
	return &MySQLController{app: application}
}

func (ctl *MySQLController) TestRead(c *gin.Context) {
	if ctl == nil || ctl.app == nil || ctl.app.MySQLDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "mysql not configured"})
		return
	}

	row := struct {
		Value int `json:"value"`
	}{}
	if err := ctl.app.MySQLDB.WithContext(c.Request.Context()).Raw("SELECT 1 AS value").Scan(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"value": row.Value})
}
