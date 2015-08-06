package auth

import (
	"fmt"
	"gopkg.in/redis.v3"
	"net/url"
	"os"
	"strconv"
	"time"
)

// redisClient returns a pointer to the Redis client.
func redisClient() *redis.Client {
	envvar := os.Getenv("REDIS_URL")
	redisURL, err := url.Parse(envvar)
	if err != nil {
		panic(fmt.Errorf("could not parse REDIS_URL `%s`", envvar))
	}
	pwd, _ := redisURL.User.Password()

	return redis.NewClient(&redis.Options{Addr: redisURL.Host, Password: pwd, DB: 0})
}

// getTokenHits returns whether the token exists and its value if so.
func getTokenHits(redisKey string, client *redis.Client) (ok bool, valueInt int) {
	value, err := client.Get(redisKey).Result()
	if err != redis.Nil && err != nil {
		panic(fmt.Errorf("getting key %s failed: %s", redisKey, err))
	}
	ok = err != redis.Nil
	if value == "" {
		valueInt = 0
	} else {
		intVal, convErr := strconv.Atoi(value)
		if convErr != nil {
			panic(fmt.Errorf("value %s from key %s could not be converted to integer: %s", value, redisKey, err))
		}
		valueInt = intVal
	}
	return
}

// incrToken increments the usage number of that token.
func incrToken(redisKey string, client *redis.Client) {
	if err := client.Incr(redisKey).Err(); err != redis.Nil && err != nil {
		panic(fmt.Errorf("incrementing key %s failed %s", redisKey, err))
	}
}

// setToken creates a new nonce and sets its expiration date and returns the expiration time.
func setToken(redisKey string, dur time.Duration, client *redis.Client) {
	if err := client.Set(redisKey, 0, dur).Err(); err != redis.Nil && err != nil {
		panic(fmt.Errorf("setting token %s failed %s", redisKey, err))
	}
}
