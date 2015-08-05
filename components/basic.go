package components

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func IndexGet(context *gin.Context) {
	context.String(http.StatusOK, "hello world!")
}

func AuthTestPost(context *gin.Context) {
	context.String(http.StatusOK, "POST success!")
}

func AuthTestPut(context *gin.Context) {
	context.String(http.StatusOK, "PUT success!")
}
