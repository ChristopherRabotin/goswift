package main

import (
	"fmt"
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/gin-gonic/gin"
	"github.com/jmcvetta/randutil"
	"gopkg.in/redis.v3"
	"net/http"
	"sync"
	"time"
)

const (
	// NonceTTL is the time to live of a Nonce token.
	NonceTTL = time.Minute * 15
	// NonceLimit is the max number of times a token can be used.
	NonceLimit = 15
)

// PerishableToken defines a header auth manager whose tokens are only valid for a short time.
type PerishableToken struct {
	redisClient *redis.Client
	*headerauth.TokenManager
}

// RedisCnx stores an instance of the redis client.
var RedisCnx = redisClient()

// CheckHeader returns the secret key from the provided access key.
func (m PerishableToken) CheckHeader(auth *headerauth.AuthInfo, req *http.Request) (err *headerauth.AuthErr) {
	auth.Secret = ""     // There is no secret key, just an access key.
	auth.DataToSign = "" // There is no data to sign.
	// Let's check if we have that token in cache, if not we'll check on Redis.
	if cached, exists := perishableCache[auth.AccessKey]; exists {
		if cached.isValid() {
			cached.Hits++
			go func() {
				incrToken(PerishableRedisKey(auth.AccessKey), m.redisClient)
			}()
		} else {
			err = &headerauth.AuthErr{401, fmt.Errorf("token expired in cache: [%s]", auth.AccessKey)}
		}
		return
	}
	// TODO: if key not in cache, get the key from redis and its value. If the key does not exist, return an error.
	// If the key exists but isn't in cache get to TTL to add it to the cache. This should be as a go routine to not cause lag.
	exists, attempts := getTokenHits(PerishableRedisKey(auth.AccessKey), m.redisClient)
	if !exists {
		// The key does not exist on Redis, let's return an error.
		err = &headerauth.AuthErr{401, fmt.Errorf("token not on Redis: [%s]", auth.AccessKey)}
		return
	}
	// Let's add this token to the cache.
	exists, ttl := getTokenTTL(PerishableRedisKey(auth.AccessKey), m.redisClient)
	if !exists {
		// The key has expired between when we checked its existence and when we got its TTL.
		err = &headerauth.AuthErr{401, fmt.Errorf("token expired on Redis: [%s]", auth.AccessKey)}
		return
	}
	// Let's store this perishable token in the cache. Because we're using it now, let's increment it locally now.
	perishable := &PerishableInfo{attempts + 1, ttl}
	if !perishable.isValid() {
		err = &headerauth.AuthErr{401, fmt.Errorf("token expired on load from Redis: [%s]", auth.AccessKey)}
		return
	}
	perishableCache[auth.AccessKey] = perishable
	go func() {
		incrToken(PerishableRedisKey(auth.AccessKey), m.redisClient)
	}()
	return
}

// Authorize sets the specified context key to the valid token (no additonals checks here, as per documentation recommendations).
func (m PerishableToken) Authorize(auth *headerauth.AuthInfo) (val interface{}, err *headerauth.AuthErr) {
	return auth.AccessKey, nil
}

// PreAbort sets the appropriate error JSON.
func (m PerishableToken) PreAbort(c *gin.Context, auth *headerauth.AuthInfo, err *headerauth.AuthErr) {
	log.Critical(c.Request.RequestURI)
	c.JSON(err.Status, StatusMsg[err.Status].JSON())
}

// NewPerishableTokenMgr returns a new PerishableToken auth manager.
func NewPerishableTokenMgr(prefix string, contextKey string) *PerishableToken {
	return &PerishableToken{RedisCnx, headerauth.NewTokenManager("Authorization", prefix, contextKey)}
}

// PerishableInfo stores perisable token information.
type PerishableInfo struct {
	Hits    int
	Expires time.Time
}

// isValid returs whether this token is still valid or not.
func (p PerishableInfo) isValid() bool {
	return p.Hits < NonceLimit && p.Expires.After(time.Now())
}

var perishableCache = make(map[string]*PerishableInfo)

// PerishableRedisKey returns the formatted Redis key for the provided perishable token.
func PerishableRedisKey(token string) string {
	return fmt.Sprintf("goswift:perishabletoken:%s", token)
}

// GetNewToken returns a JSON object which contains a new NONCE with its expiration time and the number of allowed usages.
func GetNewToken(c *gin.Context) {
	failed := true
	// Allow up to ten attempts to generate an access key.
	for iter := 0; iter < 10; iter++ {
		if token, err := randutil.AlphaStringRange(10, 10); err == nil {
			if _, inCache := perishableCache[token]; inCache {
				// If this token is already in our cache, we don't even check if it's in Redis,
				// and just ask for a new one.
				continue
			}
			if ok, _ := getTokenHits(PerishableRedisKey(token), RedisCnx); !ok {
				// We calculate the expire time prior to actually setting it so the client
				// can switch to another Nonce before it actually expires.
				expires := time.Now().Add(NonceTTL)
				perishableCache[token] = &PerishableInfo{0, expires}
				setToken(PerishableRedisKey(token), NonceTTL, RedisCnx)
				c.JSON(200, gin.H{"token": token, "expires": expires.Format(time.RFC3339), "limit": NonceLimit})
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

type AnalyticsToken struct {
	persistC chan<- *S3Persist
	wg       *sync.WaitGroup
	*PerishableToken
}

// PreAbort sets the appropriate error JSON after starting the persistence.
func (m AnalyticsToken) PreAbort(c *gin.Context, auth *headerauth.AuthInfo, err *headerauth.AuthErr) {
	m.wg.Add(1)
	c.Set(m.ContextKey(), auth.AccessKey)
	c.Set("authSuccess", false)
	m.persistC <- NewS3Persist("analytics", false, c)
	c.JSON(err.Status, StatusMsg[err.Status].JSON())
}

// PostAuth starte the persistence.
func (m AnalyticsToken) PostAuth(c *gin.Context, auth *headerauth.AuthInfo, err *headerauth.AuthErr) {
	m.wg.Add(1)
	c.Set("authSuccess", true)
	m.persistC <- NewS3Persist("analytics", false, c)
}

// NewAnalyticsTokenMgr returns a new AnalyticsToken auth manager, which is PerishableToken with S3 persistence.
func NewAnalyticsTokenMgr(prefix string, contextKey string, persistChan chan<- *S3Persist, wg *sync.WaitGroup) *AnalyticsToken {
	return &AnalyticsToken{persistChan, wg, NewPerishableTokenMgr(prefix, contextKey)}
}
