package main

import (
	"database/sql"
	"errors"
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"github.com/pmylund/go-cache"
	"io/ioutil"
	"net/http"
	"time"
)
// Better way than two different caches:
// On a regular basis, update the cache with all the valid Access Keys. If the key is not
// there, then it just is not valid, period. Only problem is that a new access key may take up
// to an hour to be valid, but that's no biggie. If needed, we can restart the server. This 
// solves the issue where an attacker generates random access keys at every request.
// TODO: look into go-cache for ticker to update the cache.

// providerCache caches the valid providers for up to a full day. They rarely cycle.
// The key of this cache is the access key and the value is an instance of ContentProviderInfo.
const (
	ProviderCacheTTL = time.Hour * 24
)

var providerCache = cache.New(ProviderCacheTTL, time.Hour*1)

// ContentProviderInfo stores basic information needed to validate or not a given content provider.
type ContentProviderInfo struct {
	id     int
	secret string
}

// invalidAccessCache stores the invalid access keys used. The avoid hitting the database if a provider
// uses an invalid access key multiple times. An invalid access key has very little chance of becoming valid.
var invalidAccessCache = cache.New(ProviderCacheTTL, time.Hour*1)

// ContentProviderMgr is an example definition of an AuthKeyManager struct.
type ContentProviderMgr struct {
	Secret string
	*headerauth.HMACManager
}

// CheckHeader returns an error if something is wrong with the header, or the auth fails (if it can fail here).
// Here should reside additional verifications on the header, or other parts of the request, if needed.
func (m ContentProviderMgr) CheckHeader(auth *headerauth.AuthInfo, req *http.Request) (err *headerauth.AuthErr) {
	if req.ContentLength == 0 || req.Body == nil {
		// This manager only support requests with a body.
		return &headerauth.AuthErr{403, errors.New("Wrong access key or signature.")}
	}

	// Let's attempt to grab the content provider information from the valid cache.
	provider, exists := providerCache.Get(auth.AccessKey)
	if !exists {
		return &headerauth.AuthErr{403, errors.New("Wrong access key or signature.")}
		
		// TODO: Put this in the function which updates the cache with all the values.
		// This access key is unknown. Let's grab it from the database.
		db := GetDBConn()
		var secret string
		dbErr := db.QueryRow("SELECT secret_key FROM apiv2_authkey WHERE disabled='f' and access_key=?", auth.AccessKey).Scan(&secret)
		switch {
		case dbErr == sql.ErrNoRows:
			// Access key does not exists. Let's cache this information.
			invalidAccessCache.Add(auth.AccessKey, struct{}{}, ProviderCacheTTL)
			return &headerauth.AuthErr{403, errors.New("Wrong access key or signature.")}
		case dbErr != nil:
			log.Critical("query failed: %s", dbErr)
			return &headerauth.AuthErr{503, errors.New("Service unavailable.")}
		}
	}

	// The access key is valid. Let's check the signature.
	body, ioErr := ioutil.ReadAll(req.Body)
	if ioErr != nil {
		log.Critical("could not read the body: %s.", ioErr)
		return &headerauth.AuthErr{503, errors.New("Service unavailable.")}
	}
	auth.Secret = provider.(*ContentProviderInfo).secret
	auth.DataToSign = string(body)
	return
}

// Authorize returns the value to store in Gin's context at ContextKey(), or an error if the auth fails.
// This is only called once the requested has been authorized to pursue, i.e. access key and signature are valid,
// so logging of success should happen here.
func (m ContentProviderMgr) Authorize(auth *headerauth.AuthInfo) (val interface{}, err *headerauth.AuthErr) {
	provider, _ := providerCache.Get(auth.AccessKey)
	val = provider.(*ContentProviderInfo).id
	return
	// TODO: Add that into the ticking function too.
	var authkeyID string
	db := GetDBConn()
	// TODO: Can't this be done in a unique SQL query?!
	db.QueryRow("SELECT id FROM apiv2_authkey WHERE disabled='f' and access_key=?", auth.AccessKey).Scan(&authkeyID)
	var cpID int
	db.QueryRow(`SELECT "apiv2_contentprovider"."id" FROM "apiv2_contentprovider" INNER JOIN "apiv2_contentprovider_authkeys" ON ( "apiv2_contentprovider"."id" = "apiv2_contentprovider_authkeys"."contentprovider_id" ) WHERE "apiv2_contentprovider_authkeys"."authkey_id" = ?`, authkeyID).Scan(&cpID)
	val = cpID
	return
}
