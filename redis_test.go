package main

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/redis.v3"
	"os"
	"testing"
)

// TestRedis tests all of features of the redis interface.
func TestRedis(t *testing.T) {
	testGoswift = true
	Convey("Without a REDIS_URL", t, func() {
		curVal := os.Getenv("REDIS_URL")
		os.Setenv("REDIS_URL", "Not a valid URL")
		So(func() { redisClient() }, ShouldPanic)
		os.Setenv("REDIS_URL", curVal)
	})
	Convey("With a valid REDIS_URL", t, func() {
		token := "testing"
		client := redisClient()
		Convey("The expected token Redis key is correct", func() {
			So(tokenToRedisKey(token), ShouldEqual, "goswift:perishabletoken:testing")
		})

		Convey("Updating a non integer Redis key fails", func() {
			if err := client.Set(tokenToRedisKey(token), "val", 0).Err(); err != redis.Nil && err != nil {
				panic(fmt.Errorf("setting token %s failed %s", token, err))
			}
			So(func() { incrToken(token, client) }, ShouldPanic)
		})

		Convey("Getting the value for a non existing key fails", func() {
			ok, _ := getTokenHits(token+"NotExist", client)
			So(ok, ShouldEqual, false)
		})
	})
}
