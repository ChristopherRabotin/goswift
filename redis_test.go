package goswift

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/redis.v3"
	"os"
	"testing"
	"time"
)

// TestRedis tests all of features of the redis interface.
func TestRedis(t *testing.T) {
	Convey("The Redis interface tests, ", t, func() {
		Convey("Without a REDIS_URL", func() {
			curVal := os.Getenv("REDIS_URL")
			os.Setenv("REDIS_URL", "//not.a.user@%66%6f%6f.com/just/a/path/also")
			So(func() { redisClient() }, ShouldPanic)
			os.Setenv("REDIS_URL", curVal)
		})

		Convey("With a valid REDIS_URL", func() {
			token := "testing"
			client := redisClient()
			Convey("The expected token Redis key is correct", func() {
				So(PerishableRedisKey(token), ShouldEqual, "goswift:perishabletoken:testing")
			})

			Convey("Getting a non integer Redis key fails", func() {
				if err := client.Set(PerishableRedisKey(token), "val", 0).Err(); err != redis.Nil && err != nil {
					panic(fmt.Errorf("setting token %s failed %s", token, err))
				}
				So(func() { getTokenHits(PerishableRedisKey(token), client) }, ShouldPanic)
			})

			Convey("Incrementing a non integer Redis key fails", func() {
				if err := client.Set(PerishableRedisKey(token), "val", 0).Err(); err != redis.Nil && err != nil {
					panic(fmt.Errorf("setting token %s failed %s", token, err))
				}
				So(func() { incrToken(PerishableRedisKey(token), client) }, ShouldPanic)
				So(func() { getTokenHits(PerishableRedisKey(token), client) }, ShouldPanic)
			})

			Convey("Getting the value for a non existing key fails", func() {
				ok, _ := getTokenHits(PerishableRedisKey(token+"NotExist"), client)
				So(ok, ShouldEqual, false)
			})

			Convey("Updating or getting an integer Redis key works", func() {
				if err := client.Set(PerishableRedisKey(token), 2, 0).Err(); err != redis.Nil && err != nil {
					panic(fmt.Errorf("setting token %s failed %s", token, err))
				}
				So(func() { incrToken(PerishableRedisKey(token), client) }, ShouldNotPanic)
				ok, val := getTokenHits(PerishableRedisKey(token), client)
				So(ok, ShouldEqual, true)
				So(val, ShouldEqual, 3) // We have incremented the value already.
			})

			Convey("Getting the TTL of a Redis key with a TTL works", func() {
				if err := client.Set(PerishableRedisKey(token), 2, time.Minute*1).Err(); err != redis.Nil && err != nil {
					panic(fmt.Errorf("setting token %s failed %s", token, err))
				}
				ok, _ := getTokenTTL(PerishableRedisKey(token), client)
				So(ok, ShouldEqual, true)
			})

			Convey("Getting the TTL of a Redis key without a TTL fails", func() {
				if err := client.Set(PerishableRedisKey(token), 2, -1).Err(); err != redis.Nil && err != nil {
					panic(fmt.Errorf("setting token %s failed %s", token, err))
				}
				ok, _ := getTokenTTL(PerishableRedisKey(token), client)
				So(ok, ShouldEqual, false)
			})

		})
	})
}
