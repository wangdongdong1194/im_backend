package main

import (
	"context"
	"log"

	httpapi "im_backend/api/http"
	"im_backend/api/socket"
	"im_backend/internal/app"

	"github.com/gin-gonic/gin"
)

func main() {
	application, err := app.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			log.Println("app close failed:", closeErr)
		}
	}()

	gin.SetMode(application.Config.GinMode)

	server := socket.NewServer(application)

	r := httpapi.NewRouter(application, server.ServeHandler(nil))
	addr := application.Config.HTTPAddr()

	log.Println("listen", addr)
	log.Fatal(r.Run(addr))
}
