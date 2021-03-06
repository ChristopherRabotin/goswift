// Package main is the main package for Goswift.
package main

import (
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"sync"
)

// testGoswift must be true when testing to avoid starting the server.
var testGoswift = false

// testS3Locations will store the list of S3 locations to delete after running the tests.
var testS3Locations []string

// log is the main go-logging logger.
var log = logging.MustGetLogger("goswift")

// persisterWg is the persister wait group, which will write to S3.
var persisterWg sync.WaitGroup

// init is ran before the main, so we'll perform the environment verifications there.
func init() {
	CheckEnvVars() // This will fail if there are env vars missing.
	ConfigureLogger()
	ConfigureRuntime()
}

// main starts all needed functions to start the server.
func main() {
	PourGin()
	persisterWg.Wait()
}

// PourGin starts pouring the gin, i.e. sets up routes and starts listening.
// This returns an engine specifically for testing purposes.
func PourGin() *gin.Engine {
	gin.SetMode(ServerMode())
	engine := gin.Default()
	engine.GET("/", IndexGet)
	// S3 persister variables
	persistChan := make(chan *S3Persist, 250)
	go S3PersistingHandler(persistChan, &persisterWg)

	// Auth managers
	perishableHA := NewPerishableTokenMgr("DecayingToken", "token")
	analyticsHA := NewAnalyticsTokenMgr("DecayingToken", "token", persistChan, &persisterWg)

	// Auth group.
	authG := engine.Group("/auth")
	authG.GET("/token", GetNewToken)
	// Auth testing group for tokens. Works on *all* methods.
	authTokenTest := authG.Group("/token/test")
	authTokenTest.Use(headerauth.HeaderAuth(perishableHA))
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, meth := range methods {
		authTokenTest.Handle(meth, "/", []gin.HandlerFunc{SuccessJSON}[0])
	}

	// Analytics group.
	analyticsG := engine.Group("/analytics")
	analyticsG.Use(headerauth.HeaderAuth(analyticsHA))
	analyticsG.PUT("/record", RecordAnalytics)
	if testGoswift {
		testS3Locations = make([]string, 0) // Allows append to assign directly to zeroth element.
	} else {
		// Starting the server.
		engine.Run(ServerConfig())
		return nil
	}
	return engine
}
