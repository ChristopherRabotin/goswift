// Package goswift is the main package for Goswift.
package goswift

import (
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

// testGoswift must be true when testing to avoid starting the server.
var testGoswift = false

// log is the main go-logging logger.
var log = logging.MustGetLogger("goswift")

// init is ran before the main, so we'll perform the environment verifications there.
func init() {
	CheckEnvVars() // This will fail if there are env vars missing.
	ConfigureLogger()
	ConfigureRuntime()
}

// main starts all needed functions to start the server.
func main() {
	PourGin()
}

// PourGin starts pouring the gin, i.e. sets up routes and starts listening.
// This returns an engine specifically for testing purposes.
func PourGin() *gin.Engine {
	gin.SetMode(ServerMode())
	engine := gin.Default()
	engine.GET("/", IndexGet)
	// Auth managers
	perishable := NewPerishableTokenMgr("DecayingToken", "token")

	// Auth group.
	authG := engine.Group("/auth")
	authG.GET("/token", GetNewToken)
	// Auth testing group for tokens. Works on *all* methods.
	authTokenTest := authG.Group("/token/test")
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
		engine.Run(ServerConfig())
		return nil
	}
	return engine
}
