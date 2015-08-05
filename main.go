package main

import (
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/Sparrho/goswift/components"
	"github.com/Sparrho/goswift/settings"
	"github.com/gin-gonic/gin"
)

// Let's have one pemanent Redis connection.
var redisCnx = redisClient()

// main starts all needed functions to start the server.
func main() {
	settings.ConfigureRuntime()
	PourGin()
}

// PourGin starts pouring the gin, i.e. sets up routes and starts listening.
func PourGin() {
	settings.CheckEnvVars() // This will fail if there are env vars missing.
	gin.SetMode(settings.ServerMode())
	engine := gin.Default()
	engine.GET("/", components.IndexGet)
	// Auth group.
	auth := engine.Group("/auth")
	auth.GET("/perishableToken", tokenGET)
	// Analytics group.
	analytics := engine.Group("/analytics")
	analytics.Use(headerauth.HeaderAuth(PerishableTokenMgr{redisCnx, headerauth.NewTokenManager("Authorization", "DecayingToken", "token")}))
	analytics.PUT("/record", components.AuthTestPut)
	// Starting the server.
	engine.Run(settings.ServerConfig())
}
