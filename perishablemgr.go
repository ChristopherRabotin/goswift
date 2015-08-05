package main

import (
	"errors"
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/gin-gonic/gin"
	"github.com/jmcvetta/randutil"
	"gopkg.in/redis.v3"
	"net/http"
	"time"
)

const (
	NONCE_TTL   = time.Minute * 15
	NONCE_LIMIT = 15
)

// PerishableTokenMgr defines a header auth manager whose tokens are only valid for a short time.
type PerishableTokenMgr struct {
	redisClient *redis.Client
	*headerauth.TokenManager
}

// CheckHeader returns the secret key from the provided access key.
func (m PerishableTokenMgr) CheckHeader(auth *headerauth.AuthInfo, req *http.Request) (err *headerauth.AuthErr) {
	auth.Secret = ""     // There is no secret key, just an access key.
	auth.DataToSign = "" // There is no data to sign.
	if ok, attempts := getTokenHits(auth.AccessKey, m.redisClient); !ok || attempts > NONCE_LIMIT {
		// Note: if we've hit the max usage limit, we just return an error and wait for Redis to
		// handle its expiration.
		err = &headerauth.AuthErr{403, errors.New("invalid token")}
	}
	return
}

// Authorize sets the specified context key to the valid token (no additonals checks here, as per documentation recommendations).
func (m PerishableTokenMgr) Authorize(auth *headerauth.AuthInfo) (val interface{}, err *headerauth.AuthErr) {
	return auth.AccessKey, nil
}

// tokenGET returns a JSON object which contains a new NONCE with its expiration time and the number of allowed usages.
func tokenGET(c *gin.Context) {
	failed := true
	// Allow up to ten attempts to generate an access key.
	for iter := 0; iter < 10; iter++ {
		if token, err := randutil.AlphaStringRange(10, 10); err == nil {
			if ok, _ := getTokenHits(token, redisCnx); !ok {
				// We calculate the expire time prior to actually setting it so the client
				// can switch to another Nonce before it actually expires.
				expires := time.Now().Add(NONCE_TTL)
				setToken(token, NONCE_TTL, redisCnx)
				c.JSON(200, gin.H{"token": token, "expires": expires.Format(time.RFC3339), "limit": NONCE_LIMIT})
				failed = false
				break
			}
		}
	}

	if failed {
		// Could not generate a valid token.
		c.JSON(503, Status503.JSON())
	}
}
