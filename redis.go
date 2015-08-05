package main

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
	redisUrl, err := url.Parse(envvar)
	if err != nil {
		panic(fmt.Errorf("could not parse REDIS_URL `%s`", envvar))
	}
	pwd, _ := redisUrl.User.Password()

	return redis.NewClient(&redis.Options{Addr: redisUrl.Host, Password: pwd, DB: 0})
}

// tokenToRedisKey returns the formatted Redis key for the provided token.
func tokenToRedisKey(token string) string {
	return fmt.Sprintf("goswift:perishabletoken:%s", token)
}

// getToken returns whether the token exists and its value if so.
func getTokenHits(token string, client *redis.Client) (ok bool, valueInt int) {
	value, err := client.Get(tokenToRedisKey(token)).Result()
	if err != redis.Nil && err != nil {
		panic(fmt.Errorf("getting key %s failed: %s", token, err))
	}
	ok = err != redis.Nil
	if value == "" {
		valueInt = 0
	} else {
		intVal, convErr := strconv.Atoi(value)
		if convErr != nil {
			panic(fmt.Errorf("value %s from key %s could not be converted to integer: %s", value, token, err))
		}
		valueInt = intVal
	}
	return
}

// incrToken increments the usage number of that token.
func incrToken(token string, client *redis.Client) {
	if err := client.Incr(tokenToRedisKey(token)).Err(); err != redis.Nil && err != nil {
		panic(fmt.Errorf("incrementing token %s failed %s", token, err))
	}
}

// setToken creates a new nonce and sets its expiration date and returns the expiration time.
func setToken(token string, dur time.Duration, client *redis.Client) {
	if err := client.Set(tokenToRedisKey(token), 0, dur).Err(); err != redis.Nil && err != nil {
		panic(fmt.Errorf("setting token %s failed %s", token, err))
	}
}
