package main

import (
	"github.com/gin-gonic/gin"
)

type DefaultStatus int

const (
	Status503 DefaultStatus = 1 + iota
	Status400
	Status401
	Status403
)

var json = [...]interface{}{ // This is in the same order as the DefaultStatus const.
	gin.H{"error": "service unavailable"},
	gin.H{"error": "client error"},
	gin.H{"error": "unauthorized"},
	gin.H{"error": "forbidden"},
	gin.H{"error": "not found"},
}

// JSON returns the default JSON error for the provided status.
func (status DefaultStatus) JSON() interface{} {
	return json[status-1]
}
