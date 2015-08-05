package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func IndexGet(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, "http://www.sparrho.com/")
}

func SuccessJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"method": c.Request.Method})
}
