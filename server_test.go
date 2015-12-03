package main

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

// TestServer tests stuff from runtime.go
func TestServer(t *testing.T) {
	Convey("The Server configuration tests, ", t, func() {

		Convey("Playing with SERVER_ADDR and SERVER_PORT", func() {
			var serverConfs = []struct {
				addr string
				port string
				expt string
			}{
				{"", "", fmt.Sprintf(":%s", DefaultPort)},
				{"", "1025", ":1025"},
				{"127.0.0.1", "", fmt.Sprintf("127.0.0.1:%s", DefaultPort)},
				{"127.0.0.1", "1025", "127.0.0.1:1025"},
			}
			for _, conf := range serverConfs {
				os.Setenv("SERVER_ADDR", conf.addr)
				os.Setenv("SERVER_PORT", conf.port)
				So(ServerConfig(), ShouldEqual, conf.expt)
			}
		})

		envvars := []string{"SERVER_MODE", "SERVER_PORT"}
		for i := range envvars {
			envvar := envvars[i]
			Convey(fmt.Sprintf("Unsetting %s", envvar), func() {
				curVal := os.Getenv(envvar)
				os.Unsetenv(envvar)
				So(func() { ServerMode() }, ShouldNotPanic)
				So(func() { ServerConfig() }, ShouldNotPanic)
				os.Setenv(envvar, curVal)
			})
		}
	})
}
