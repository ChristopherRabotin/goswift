package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/op/go-logging"
	"os"
	"runtime"
	"strconv"
)

// CheckEnvVars checks that all the environment variables required are set, without checking their value. It will panic if one is missing.
func CheckEnvVars() {
	envvars := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_STORAGE_BUCKET_NAME", "REDIS_URL", "DATABASE_URL"}
	for _, envvar := range envvars {
		if os.Getenv(envvar) == "" {
			panic(fmt.Errorf("environment variable `%s` is missing or empty", envvar))
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
	log.Info("Running with %d CPUs.\n", useNumCPUs)
}

// ConfigureLogger configures the default logger (named "gofetch").
func ConfigureLogger() {
	// From https://github.com/op/go-logging/blob/master/examples/example.go.
	logFormat := logging.MustStringFormatter("%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level}%{color:reset} %{message}")
	logging.SetBackend(logging.NewBackendFormatter(logging.NewLogBackend(os.Stderr, "", 0), logFormat))
	// Let's grab the log level from the environment, or set it to INFO.
	envlvl := os.Getenv("LOG_LEVEL")
	if envlvl != "" {
		lvl, err := logging.LogLevel(envlvl)
		if err != nil {
			lvl = logging.INFO
		}
		log.Notice("Set logging level to %s.\n", lvl)
		logging.SetLevel(lvl, "")
	} else {
		log.Notice("No log level defined in environment. Defaulting to INFO.\n")
		logging.SetLevel(logging.INFO, "")
	}
}

// GetDBConn returns a database connection. Note that database/sql handles a connection pool by itself.
func GetDBConn() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(fmt.Errorf("could not connect to database `%s`", err))
	}
	return db
}
