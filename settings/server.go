package settings

import (
	"fmt"
	"strconv"
	"syscall"
)

const (
	DEFAULT_PORT        = "1024" // The default port is a string because we return a string anyway.
	DEFAULT_SERVER_MODE = "debug"
)

func ServerConfig() string {
	addr, _ := syscall.Getenv("SERVER_ADDR")
	var port string
	if portStr, ok := syscall.Getenv("SERVER_PORT"); !ok {
		port = DEFAULT_PORT
	} else {
		portUInt, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil || portUInt > 65535 {
			log.Notice("Invalid port %s, using %s instead.", portStr, DEFAULT_PORT)
			port = DEFAULT_PORT
		} else {
			port = portStr
		}
	}
	return fmt.Sprintf("%s:%s", addr, port)
}

func ServerMode() string {
	mode, ok := syscall.Getenv("SERVER_MODE")
	if !ok {
		mode = DEFAULT_SERVER_MODE
	}
	return mode
}
