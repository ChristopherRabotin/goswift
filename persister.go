package goswift

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"sync"
	"time"
)

const (
	rootPath    = "/goswift"
	storePath   = "/incoming"
	indexFolder = "/index/sha384_checksum"
	dataFolder  = "/sha384_data"
)

// S3Persister defines how and what is persisted to S3.
type S3Persister interface {
	setContext(*gin.Context) // sets the context for this persister instance.
	indexed() bool           // Whether or not this persister is indexed to avoid duplication.
	location() string        // location on S3 for this persisted data.
	serialize() string       // serialized data to store on S3.
	checksum() string        // checksum of the context, used only if this is an index persister.
}

// S3Persist implements of an S3Persister.
type S3Persist struct {
	path    string
	isIndxd bool
	c       *gin.Context
	body    string
}

func (p S3Persist) setContext(c *gin.Context) {
	p.c = c
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		panic("could not read body")
	}
	p.body = string(body)
}

func (p S3Persist) indexed() bool {
	return p.isIndxd
}

// serialize returns what is to be serialized and saved to S3.
func (p S3Persist) serialize() string {
	return p.body
}

// checksum returns the checksum of this analytics event, which is not used for analytics.
func (p S3Persist) checksum() string {
	hash := sha512.New384()
	hash.Write([]byte(p.body))
	return hex.EncodeToString(hash.Sum(nil))
}

// location returns the location where this context will be persisted.
func (p S3Persist) location() string {
	loc := rootPath + storePath
	if testGoswift {
		loc += "/test"
	}
	loc += dataFolder + "/" + p.path + "/"
	now := time.Now().UTC()
	y, m, d := now.Date()
	successIft, _ := p.c.Get("authSuccess")
	if success, ok := successIft.(bool); ok {
		if success {
			loc += "valid/"
		} else {
			loc += "invalid/"
		}
	} else {
		loc += "unknown/"
	}
	loc += fmt.Sprintf("%d/%d/%d/%02d00/", y, m, d, now.Hour())
	accessKeyItf, _ := p.c.Get("token")
	if accessKey, ok := accessKeyItf.(string); ok {
		loc += accessKey
	} else {
		loc += "noAccessKeyFound"
	}
	randStr, _ := randutil.AlphaStringRange(20, 20)

	return loc + p.checksum() + "_" + randStr
}

func NewS3Persist(path string, indexed bool, c *gin.Context) *S3Persist {
	return &S3Persist{path: path, isIndxd: indexed, c: c}
}

// storeOnS3 stores information from the contextChan onto S3.
// It expects there to be a context key called "valid", storing a boolean, and "token" storing an access key.
// For example, if valid is true, token is "a1d5r9", the context will be stored in
// /goswift/incoming/accepted/{endpoint}/{date in ISO format rounded to the lowest hour}/a1d5r9_{sha384 of the data}_{random token}
// This file will then contain the body provided in the context.
func persistToS3(persistChan <-chan *S3Persister, wg *sync.WaitGroup) {
	// Round to the hour: fmt.Println(time.Now().UTC().Format("2006-01-02T150000Z07:00"))
}

// indexOnS3 works in a similar fashion to storeOnS3. However, it will only store the new context to S3, and index it if it was not new.
// This index is based on the SHA384 of the data. Anything which was not found on S3 index will be added to the newChan for further processing.
// For example, the newChan will then forward the request to the Content API. This will massively reduce the load on the ContentAPI, and only
// attempt to add information which we know has not been encountered before.
func indexOnS3(persistChan <-chan *S3Persister, newChan chan<- *gin.Context, wg *sync.WaitGroup) {

}
