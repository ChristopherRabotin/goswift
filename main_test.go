package goswift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jmcvetta/randutil"
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
	testSettings := map[string]string{"MAX_CPUS": "1", "AWS_STORAGE_BUCKET_NAME": "sparrho-content",
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

		Convey("Perishable Tokens can be generated and stored on this instance", func() {
			req := performRequest(e, "GET", "/auth/token", nil, nil)
			So(req.Code, ShouldEqual, 200)
			var tok TokenResponse
			json.Unmarshal(req.Body.Bytes(), &tok)

			So(tok.Limit, ShouldEqual, NonceLimit)
			expirationValid := tok.Expires.Sub(time.Now()) < NonceTTL
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
				remaining := NonceLimit - tok.NumUsed
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

		Convey("Perishable Tokens can be retrieved from redis", func() {
			req := performRequest(e, "GET", "/auth/token", nil, nil)
			So(req.Code, ShouldEqual, 200)
			var tok TokenResponse
			json.Unmarshal(req.Body.Bytes(), &tok)

			So(tok.Limit, ShouldEqual, NonceLimit)
			expirationValid := tok.Expires.Sub(time.Now()) < NonceTTL
			So(expirationValid, ShouldEqual, true)
			_, tokenInCache := perishableCache[tok.Token]
			So(tokenInCache, ShouldEqual, true)
			delete(perishableCache, tok.Token)
			_, tokenInCache = perishableCache[tok.Token]
			So(tokenInCache, ShouldEqual, false)

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
				remaining := NonceLimit - tok.NumUsed
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

			Convey("And the token could have timed out on Redis", func() {
				// Let's update this token on Redis to an invalid number of hits.
				redisClient().Set(PerishableRedisKey(tok.Token), NonceLimit+1, 0)

				headers := make(map[string][]string)
				headers["Authorization"] = []string{"DecayingToken " + tok.Token}

				req := performRequest(e, "GET", "/auth/token/test/", headers, nil)
				var resp ErrorResponse
				json.Unmarshal(req.Body.Bytes(), &resp)

				So(req.Code, ShouldEqual, 401)
				So(resp.Error, ShouldEqual, "unauthorized")

			})

			Convey("And the token could have reached max hits on Redis", func() {
				// Let's update this token on Redis to an invalid number of hits.
				redisClient().Set(PerishableRedisKey(tok.Token), NonceLimit+1, time.Minute*5)

				headers := make(map[string][]string)
				headers["Authorization"] = []string{"DecayingToken " + tok.Token}

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
			RedisCnx.Del(PerishableRedisKey(invalidToken))
			headers["Authorization"] = []string{"DecayingToken " + invalidToken}
			for _, meth := range methods {
				req := performRequest(e, meth, "/auth/token/test/", headers, nil)
				var resp ErrorResponse
				json.Unmarshal(req.Body.Bytes(), &resp)

				So(req.Code, ShouldEqual, 401)
				So(resp.Error, ShouldEqual, "unauthorized")

			}
		})

		Convey("Analytics endpoint works as expected", func() {

			//Let's first grab a token.
			req := performRequest(e, "GET", "/auth/token", nil, nil)
			So(req.Code, ShouldEqual, 200)
			var tok TokenResponse
			json.Unmarshal(req.Body.Bytes(), &tok)

			So(tok.Limit, ShouldEqual, NonceLimit)
			So(tok.Expires.Sub(time.Now()) < NonceTTL, ShouldEqual, true)
			Convey("By failing on all methods but PUT", func() {
				headers := make(map[string][]string)
				headers["Authorization"] = []string{"DecayingToken " + tok.Token}
				for _, meth := range methods {
					req := performRequest(e, meth, "/analytics/record", headers, nil)
					tok.NumUsed++ // Incrementing the number of times this one was used to confirm it will expire later.
					var resp SuccessResponse
					json.Unmarshal(req.Body.Bytes(), &resp)
					if meth == "PUT" {
						So(req.Code, ShouldEqual, 202)
						So(req.Body.String(), ShouldEqual, "")
					} else {
						So(req.Code, ShouldEqual, 404)
					}
				}
			})

			Convey("By failing if the token is invalid", func() {
				headers := make(map[string][]string)
				headers["Authorization"] = []string{"DecayingToken InvalidToken"}

				for _, meth := range methods {
					req := performRequest(e, meth, "/analytics/record", headers, NewAnalyticsEvent().JSON())
					tok.NumUsed++ // Incrementing the number of times this one was used to confirm it will expire later.
					var resp SuccessResponse
					json.Unmarshal(req.Body.Bytes(), &resp)
					if meth == "PUT" {
						So(req.Code, ShouldEqual, 401)
					} else {
						So(req.Code, ShouldEqual, 404)
					}
				}
			})

			Convey("PUT requests persist the data on S3", func() {
				Convey("If the token is valid", func() {
					headers := make(map[string][]string)
					headers["Authorization"] = []string{"DecayingToken " + tok.Token}

					req := performRequest(e, "PUT", "/analytics/record", headers, NewAnalyticsEvent().JSON())
					tok.NumUsed++ // Incrementing the number of times this one was used to confirm it will expire later.
					var resp SuccessResponse
					json.Unmarshal(req.Body.Bytes(), &resp)
					So(req.Code, ShouldEqual, 202)

				})
			})
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

// AnalyticsJSON stores an analytics event as JSON.
type AnalyticsJSON struct {
	user_agent string
	referer    string
	km_ai      string
	session_id string
	url_path   string
	client_ip  string
}

func (e AnalyticsJSON) JSON() (buf io.Reader) {
	jsonBody, _ := json.Marshal(&e)
	buf = bytes.NewBuffer(jsonBody)
	return
}

func NewAnalyticsEvent() *AnalyticsJSON {
	randToken, _ := randutil.AlphaStringRange(10, 10)
	return &AnalyticsJSON{client_ip: "127.0.0.1", km_ai: randToken[0:7], session_id: "session_" + randToken[5:10],
		user_agent: "Mozilla/5.0 (X11; Linux x86_64; rv:39.0) Gecko/20100101 Firefox/39.0", url_path: "http://sparrho.com/awesome/link"}
}
