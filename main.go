package main

import (
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/Sparrho/goswift/settings"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

// testGoswift must be true when testing to avoid starting the server.
var testGoswift = false

// Let's have one pemanent Redis connection.
var redisCnx = redisClient()

// log is the main go-logging logger.
var log = logging.MustGetLogger("goswift")

// init is ran before the main, so we'll perform the environment verifications there.
func init() {
	settings.CheckEnvVars() // This will fail if there are env vars missing.
	settings.ConfigureLogger()
	settings.ConfigureRuntime()
}

// main starts all needed functions to start the server.
func main() {
	PourGin()
}

// PourGin starts pouring the gin, i.e. sets up routes and starts listening.
// This returns an engine specifically for testing purposes.
func PourGin() *gin.Engine {
	gin.SetMode(settings.ServerMode())
	engine := gin.Default()
	engine.GET("/", IndexGet)
	// Auth managers
	perishable := PerishableTokenMgr{redisCnx, headerauth.NewTokenManager("Authorization", "DecayingToken", "token")}

	// Auth group.
	auth := engine.Group("/auth")
	auth.GET("/token", tokenGET)
	// Auth testing group for tokens. Works on *all* methods.
	authTokenTest := auth.Group("/token/test")
	authTokenTest.Use(headerauth.HeaderAuth(perishable))
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, meth := range methods {
		authTokenTest.Handle(meth, "/", []gin.HandlerFunc{SuccessJSON}[0])
	}

	// Analytics group.
	analytics := engine.Group("/analytics")
	analytics.Use(headerauth.HeaderAuth(perishable))
	analytics.PUT("/record", SuccessJSON)
	if !testGoswift {
		// Starting the server.
		engine.Run(settings.ServerConfig())
		return nil
	}
	return engine
}
