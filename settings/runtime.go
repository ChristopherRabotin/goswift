package settings

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
)

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_STORAGE_BUCKET_NAME", "REDIS_URL"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty,", envvar))
		}
	}
}

// ConfigureRuntime configures the server runtime, including the number of CPUs to use.
func ConfigureRuntime() {
	// Note that we're using os instead of syscall because we'll be parsing the int anyway, so there is no need to check if the envvar was found.
	useNumCPUsStr := os.Getenv("MAX_CPUS")
	useNumCPUsInt, err := strconv.ParseInt(useNumCPUsStr, 10, 0)
	useNumCPUs := int(useNumCPUsInt)
	if err != nil {
		useNumCPUs = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(useNumCPUs)
	log.Printf("Running with %d CPUs.\n", useNumCPUs)
}
