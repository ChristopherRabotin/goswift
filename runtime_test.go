package goswift

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

// TestRuntime tests stuff from runtime.go
func TestRuntime(t *testing.T) {
	Convey("The Runtime configuration tests, ", t, func() {
		envvars := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_STORAGE_BUCKET_NAME", "REDIS_URL"}
		for i := range envvars {
			envvar := envvars[i]
			Convey(fmt.Sprintf("Without %s", envvar), func() {
				curVal := os.Getenv(envvar)
				os.Unsetenv(envvar)
				So(func() { CheckEnvVars() }, ShouldPanic)
				os.Setenv(envvar, curVal)
			})
		}

		Convey("Without MAX_CPUS", func() {
			envvar := "MAX_CPUS"
			curVal := os.Getenv(envvar)
			os.Unsetenv(envvar)
			So(func() { ConfigureRuntime() }, ShouldNotPanic)
			os.Setenv(envvar, curVal)
		})

		invalidLogLevels := []string{"D3BUG", ""}
		for i := range invalidLogLevels {
			envvar := "LOG_LEVEL"
			envval := invalidLogLevels[i]
			Convey(fmt.Sprintf("With %s to %s", envvar, envval), func() {
				curVal := os.Getenv(envvar)
				os.Setenv(envvar, envval)
				So(func() { ConfigureLogger() }, ShouldNotPanic)
				os.Setenv(envvar, curVal)
			})
		}
	})
}
