package main

import (
	"database/sql"
	"errors"
	"github.com/ChristopherRabotin/gin-contrib-headerauth"
	"io/ioutil"
	"net/http"
)

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

	// Let's check that this access key exists.
	db := GetDBConn()
	var secret string
	dbErr := db.QueryRow("SELECT secret_key FROM apiv2_authkey WHERE disabled='f' and access_key=?", auth.AccessKey).Scan(&secret)
	switch {
	case dbErr == sql.ErrNoRows:
		return &headerauth.AuthErr{403, errors.New("Wrong access key or signature.")}
	case dbErr != nil:
		log.Critical("query failed: %s", dbErr)
		return &headerauth.AuthErr{503, errors.New("Service unavailable.")}
	}

	// The access key is valid. Let's check the signature.
	body, ioErr := ioutil.ReadAll(req.Body)
	if ioErr != nil {
		log.Critical("could not read the body: %s.", ioErr)
		return &headerauth.AuthErr{503, errors.New("Service unavailable.")}
	}
	auth.Secret = secret
	auth.DataToSign = string(body)
	return
}

// Authorize returns the value to store in Gin's context at ContextKey(), or an error if the auth fails.
// This is only called once the requested has been authorized to pursue, i.e. access key and signature are valid,
// so logging of success should happen here.
func (m ContentProviderMgr) Authorize(auth *headerauth.AuthInfo) (val interface{}, err *headerauth.AuthErr) {
	var authkeyID string
	db := GetDBConn()
	// TODO: Can't this be done in a unique SQL query?!
	db.QueryRow("SELECT id FROM apiv2_authkey WHERE disabled='f' and access_key=?", auth.AccessKey).Scan(&authkeyID)
	var cpID int
	db.QueryRow(`SELECT "apiv2_contentprovider"."id" FROM "apiv2_contentprovider" INNER JOIN "apiv2_contentprovider_authkeys" ON ( "apiv2_contentprovider"."id" = "apiv2_contentprovider_authkeys"."contentprovider_id" ) WHERE "apiv2_contentprovider_authkeys"."authkey_id" = ?`, authkeyID).Scan(&cpID)
	val = cpID
	return
}
