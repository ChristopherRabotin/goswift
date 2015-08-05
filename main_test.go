package main

import (
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestSwift tests all of GoSwift features.
func TestSwift(t *testing.T) {
	testGoswift = true
	// Setting some environment variables.
	testSettings := map[string]string{"MAX_CPUS": "1", "AWS_STORAGE_BUCKET_NAME": "sparrho-static-staging",
		"SERVER_MODE": "debug", "LOG_LEVEL": "DEBUG"}
	for env, val := range testSettings {
		err := os.Setenv(env, val)
		if err != nil {
			panic(fmt.Errorf("could not set %s to %s", env, val))
		}
		log.Debug("Set envvar %s to %s.", env, val)
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	Convey("With Goswift", t, func() {

		// TokenResponse stores unmarshaled responses from GET /auth/token.
		type TokenResponse struct {
			Expires time.Time
			Limit   int
			Token   string
			NumUsed int
		}

		type SuccessResponse struct {
			Method string
		}

		type ErrorResponse struct {
			Error string
		}

		ConfigureLogger()
		ConfigureRuntime()
		e := PourGin()
		Convey("GET root redirects", func() {
			req := performRequest(e, "GET", "/", nil, nil)
			So(req.Code, ShouldEqual, 303)
		})

		Convey("Perishable Tokens can be generated", func() {
			req := performRequest(e, "GET", "/auth/token", nil, nil)
			So(req.Code, ShouldEqual, 200)
			var tok TokenResponse
			json.Unmarshal(req.Body.Bytes(), &tok)

			So(tok.Limit, ShouldEqual, NONCE_LIMIT)
			expirationValid := tok.Expires.Sub(time.Now()) < NONCE_TTL
			So(expirationValid, ShouldEqual, true)

			Convey("And can be used on the auth test endpoint for all methods until its limit", func() {
				headers := make(map[string][]string)
				headers["Authorization"] = []string{"DecayingToken " + tok.Token}
				for _, meth := range methods {
					req := performRequest(e, meth, "/auth/token/test/", headers, nil)
					tok.NumUsed++ // Incrementing the number of times this one was used to confirm it will expire later.
					var resp SuccessResponse
					json.Unmarshal(req.Body.Bytes(), &resp)

					So(req.Code, ShouldEqual, 200)
					So(resp.Method, ShouldEqual, meth)
				}

				// Let's check that the token will perish after the limit is hit.
				remaining := NONCE_LIMIT - tok.NumUsed
				for i := 0; i < remaining; i++ {
					So(performRequest(e, "GET", "/auth/token/test/", headers, nil).Code, ShouldEqual, 200)
					tok.NumUsed++
				}

				req := performRequest(e, "GET", "/auth/token/test/", headers, nil)
				var resp ErrorResponse
				json.Unmarshal(req.Body.Bytes(), &resp)

				So(req.Code, ShouldEqual, 401)
				So(resp.Error, ShouldEqual, "unauthorized")

			})
		})

		Convey("Invalid Persishable Tokens fail on the test endpoints fails for all methods", func() {
			headers := make(map[string][]string)
			invalidToken := "someinvalidtoken"
			// Let's make sure we remove this from redis.
			redisCnx.Del(tokenToRedisKey(invalidToken))
			headers["Authorization"] = []string{"DecayingToken " + invalidToken}
			for _, meth := range methods {
				req := performRequest(e, meth, "/auth/token/test/", headers, nil)
				log.Info("%s", req.Body.Bytes())
				var resp ErrorResponse
				json.Unmarshal(req.Body.Bytes(), &resp)

				So(req.Code, ShouldEqual, 401)
				So(resp.Error, ShouldEqual, "unauthorized")

			}
		})

	})
}

// performRequest is a helper to test requests (taken from the gin-contrib-headerauth tests).
func performRequest(r http.Handler, method string, path string, headers map[string][]string, body io.Reader) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	req.Header = headers
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
