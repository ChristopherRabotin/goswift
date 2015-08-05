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

var jsonStatus = [...]interface{}{ // This is in the same order as the DefaultStatus const.
	gin.H{"error": "service unavailable"},
	gin.H{"error": "client error"},
	gin.H{"error": "unauthorized"},
	gin.H{"error": "forbidden"},
	gin.H{"error": "not found"},
}

var statusMsg = map[int]DefaultStatus{503: Status503, 400: Status400, 401: Status401, 403: Status403}

// JSON returns the default JSON error for the provided status.
func (status DefaultStatus) JSON() interface{} {
	return jsonStatus[status-1]
}
