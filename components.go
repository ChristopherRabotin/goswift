package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// IndexGet redirects to sparrho.com
func IndexGet(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, "http://www.sparrho.com/")
}

// SuccessJSON returns a JSON saying which method was used.
func SuccessJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"method": c.Request.Method})
}
