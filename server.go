package goswift

import (
	"fmt"
	"strconv"
	"syscall"
)

const (
	// DefaultPort is the default port on which the Gin server runs.
	DefaultPort       = "1024"
	// DefaultServerMode is the default server mode for Gin.
	DefaultServerMode = "debug"
)

// ServerConfig returns the Gin server config as per environment or default.
func ServerConfig() string {
	addr, _ := syscall.Getenv("SERVER_ADDR")
	var port string
	if portStr, ok := syscall.Getenv("SERVER_PORT"); !ok {
		port = DefaultPort
	} else {
		portUInt, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil || portUInt > 65535 {
			log.Notice("Invalid port \"%s\", using %s instead.", portStr, DefaultPort)
			port = DefaultPort
		} else {
			port = portStr
		}
	}
	return fmt.Sprintf("%s:%s", addr, port)
}

// ServerMode returns the Gin server mode as per environment or default.
func ServerMode() string {
	mode, ok := syscall.Getenv("SERVER_MODE")
	if !ok {
		mode = DefaultServerMode
	}
	return mode
}
