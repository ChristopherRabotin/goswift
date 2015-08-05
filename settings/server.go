package settings

import (
	"fmt"
	"strconv"
	"syscall"
)

const (
	DEFAULT_PORT        = 1024
	DEFAULT_SERVER_MODE = "debug"
)

func ServerConfig() string {
	addr, _ := syscall.Getenv("SERVER_ADDR")
	var port uint16
	if portStr, ok := syscall.Getenv("SERVER_PORT"); !ok {
		port = DEFAULT_PORT
	} else {
		tmpport, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			port = DEFAULT_PORT
		} else {
			port = uint16(tmpport)
		}
	}
	return fmt.Sprintf("%s:%d", addr, port)
}

func ServerMode() string {
	mode, ok := syscall.Getenv("SERVER_MODE")
	if !ok {
		mode = DEFAULT_SERVER_MODE
	}
	return mode
}
