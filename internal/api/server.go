package api

import (
	"net/http"

	"github.com/asianchinaboi/backendserver/internal/api/routes"
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/handlers"
)

func StartServer() *http.Server {
	r := gin.New()
	routes.PrepareRoutes(r)
	server := &http.Server{ //server settings
		Addr: config.Config.Server.Host + ":" + config.Config.Server.Port,
		//prevents ddos attacks
		WriteTimeout: config.Config.Server.Timeout.Write,
		ReadTimeout:  config.Config.Server.Timeout.Read,
		IdleTimeout:  config.Config.Server.Timeout.Idle,
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"content-type", "Authorization", ""}), //took some time to figure out middleware problem
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"}),
			handlers.AllowCredentials(),
		)(r),
	}
	return server
}
